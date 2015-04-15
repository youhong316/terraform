resource "aws_instance" "foo" {
    count = 2

    provisioner "shell" {
        command = "echo ${aws_instance.foo.1.id}"
    }
}
