bluebox
=======

`bluebox` is intended to fast build a low overhead environment to be able to run tests against [Linux kernel](https://kernel.org/) APIs like [netlink](https://man7.org/linux/man-pages/man7/netlink.7.html) or [ebpf](https://man7.org/linux/man-pages/man2/bpf.2.html). It embeds given statically linked executables into the resulting archive. In a virtual environment with this archive the embedded executables will be executed in a sequential order.
`bluebox` does not provide a shell or other executables.

## Installation

```
$ go install github.com/florianl/bluebox@latest
```

## Example usage

In the following example `qemu-system-x86_64` is required to start the virtual environment. For the kernel image a self compiled kernel or a prepared kernel like they are offered by [github.com/cilium/ci-kernels](https://github.com/cilium/ci-kernels) can be used. If the kernel is compiled for a different architecture, then a different version of `qemu` is required as well `bluebox` also need to know about the target architecture.

```
  # Generate a very basic initial ramdisk
$ bluebox -o my-initramfs.cpio
  # Boot a kernel in a virtual environment with the generated archive
$ qemu-system-x86_64 -m 4096 -kernel my-linux.bz -initrd my-initramfs.cpio
```

A more detailed example of how `bluebox` can be used is given in [EXAMPLE.md](https://github.com/florianl/bluebox/blob/main/EXAMPLE.md).

## Requirements

A version of Go that is [supported by upstream](https://golang.org/doc/devel/release.html#policy)

## Future ideas

- use [text/template](https://pkg.go.dev/text/template) to write the init program instead of hardcoding it
- make mounts that are executed by init configurable
- check if the given executables are statically linked before embedding them

## Similar projects

- [u-root](https://github.com/u-root/u-root)
- [busybox](https://www.busybox.net)
- [virtme](https://git.kernel.org/pub/scm/utils/kernel/virtme/virtme.git/)
