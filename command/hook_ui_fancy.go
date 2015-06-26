package command

import (
	"fmt"
	"sort"
	"sync"

	"github.com/gizak/termui"
	"github.com/hashicorp/terraform/terraform"
)

// UiHookFancy is the UiHook that implements the "fancy" UI: a UI that
// uses graphical elements in the terminal to output information.
type UiHookFancy struct {
	terraform.NilHook

	// Internal attributes that should be modified under a lock
	lock      sync.Mutex
	active    map[string]string
	errored   []string
	completed []string

	// Internal UI elements that shouldn't be touched
	grid          *termui.Grid
	listActive    *termui.List
	listCompleted *termui.List
	listErrored   *termui.List
	quitCh        chan struct{}
	redrawCh      chan struct{}
}

func (h *UiHookFancy) Init() error {
	h.active = make(map[string]string)

	// Initialize the UI
	termuiOnce.Do(termuiInit)
	if termuiErr != nil {
		return termuiErr
	}

	// We want colors
	termui.UseTheme("helloworld")

	// Build the list for completed resources
	h.listCompleted = termui.NewList()
	h.listCompleted.HasBorder = true
	h.listCompleted.Border.Label = "Completed Resources"

	// Build the list for actively changing resources
	h.listActive = termui.NewList()
	h.listActive.HasBorder = true
	h.listActive.Border.Label = "Active Resources"

	// Build the list for completed resources
	h.listErrored = termui.NewList()
	h.listErrored.HasBorder = true
	h.listErrored.Border.Label = "Errored Resources"

	// Start building the grid that we're going to use
	h.grid = termui.NewGrid()
	{
		row := termui.NewRow(
			termui.NewCol(6, 0, h.listActive, h.listErrored),
			termui.NewCol(6, 0, h.listCompleted),
		)
		h.grid.AddRows(row)
	}

	// Recalculate sizes
	h.recalcSizes()

	// Make the channels and start the render goroutine
	h.quitCh = make(chan struct{})
	h.redrawCh = make(chan struct{})
	go h.run()
	h.redraw()

	return nil
}

func (h *UiHookFancy) Close() error {
	close(h.quitCh)
	termui.Close()
	return nil
}

func (h *UiHookFancy) PreApply(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (terraform.HookAction, error) {
	id := n.HumanId()

	op := uiResourceModify
	if d.Destroy {
		op = uiResourceDestroy
	} else if s.ID == "" {
		op = uiResourceCreate
	}

	var text string
	switch op {
	case uiResourceModify:
		text = "Modifying..."
	case uiResourceDestroy:
		text = "Destroying..."
	case uiResourceCreate:
		text = "Creating..."
	case uiResourceUnknown:
		return terraform.HookActionContinue, nil
	}

	h.lock.Lock()
	h.active[id] = text
	h.lock.Unlock()

	h.redraw()
	return terraform.HookActionContinue, nil
}

func (h *UiHookFancy) PostApply(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState,
	applyerr error) (terraform.HookAction, error) {
	id := n.HumanId()

	h.lock.Lock()

	// Delete the element from the active list
	delete(h.active, id)

	// If it errored put it in the error list, otherwise put it in the completed
	if applyerr != nil {
		h.errored = append(h.errored, id)
	} else {
		h.completed = append(h.completed, id)
	}

	// Unlock and redraw
	h.lock.Unlock()
	h.redraw()

	return terraform.HookActionContinue, nil
}

func (h *UiHookFancy) PreDiff(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState) (terraform.HookAction, error) {
	return terraform.HookActionContinue, nil
}

func (h *UiHookFancy) PreProvision(
	n *terraform.InstanceInfo,
	provId string) (terraform.HookAction, error) {
	return terraform.HookActionContinue, nil
}

func (h *UiHookFancy) ProvisionOutput(
	n *terraform.InstanceInfo,
	provId string,
	msg string) {
}

func (h *UiHookFancy) PreRefresh(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState) (terraform.HookAction, error) {
	return terraform.HookActionContinue, nil
}

// recalcSizes recalculates all the heights/widths and offsets of everything
// and re-aligns the grid. This should be called anytime a terminal resize
// happens.
func (h *UiHookFancy) recalcSizes() {
	totalHeight := termui.TermHeight()

	h.listActive.Height = totalHeight / 2
	h.listErrored.Height = totalHeight / 2
	h.listCompleted.Height = h.listActive.Height * 2

	h.grid.Width = termui.TermWidth()
	h.grid.Align()
}

// redraw triggers a UI redraw.
func (h *UiHookFancy) redraw() {
	// Lock since we're accessing data
	h.lock.Lock()

	// Build the list of active items
	{
		items := make([]string, 0, len(h.active))
		for k, v := range h.active {
			items = append(items, fmt.Sprintf("%s: %s", k, v))
		}
		sort.Strings(items)

		h.listActive.Items = items
	}

	sort.Strings(h.completed)
	sort.Strings(h.errored)
	h.listCompleted.Items = h.completed
	h.listErrored.Items = h.errored

	// Unlock so we can redraw
	h.lock.Unlock()

	// Draw!
	h.redrawCh <- struct{}{}
}

// run is the run loop that lives forever (until quitCh is closed) and
// redraws the UI whenever an event happens.
func (h *UiHookFancy) run() {
	eventCh := termui.EventCh()
	for {
		select {
		case <-h.quitCh:
			return
		case <-h.redrawCh:
			// We have to lock here since we don't want the list data
			// to change underneath us.
			h.lock.Lock()
			termui.Render(h.grid)
			h.lock.Unlock()
		case evt := <-eventCh:
			switch evt.Type {
			case termui.EventResize:
				// Terminal resize. Recalculate the width our grid and
				// do a redraw. We send the redraw in a goroutine so it
				// never blocks this loop
				h.recalcSizes()
				go h.redraw()
			}
		}
	}
}

// termuiOnce makes sure we only initialize the termui library once,
// and termuiErr tracks whether there was an error initializing it.
var termuiOnce sync.Once
var termuiErr error

func termuiInit() {
	termuiErr = termui.Init()
}
