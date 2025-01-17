## Overview

This debugging guide continues at the point where [EXAMPLE.md](https://github.com/florianl/bluebox/blob/main/EXAMPLE.md) stopped. At this point `bluebox` created an archive called `initramfs.cpio` that we want debug further.

## Required software

In this guide the following software is used.

- [qemu](https://www.qemu.org/)
- [gdb](https://www.sourceware.org/gdb/)
- [cpio](https://www.gnu.org/software/cpio/manual/) (optional)

## Extract files from the initramfs archive

As an optional step one can extract embeded files from the archive.

```
$ cpio -iv < /tmp/initramfs.cpio
```

This command will extract at least the `init` executable from `initramfs.cpio` that is dynamically created when using `busybox`. If `busybox` was instructed to embedd more executables or files into the archive, these will be extracted from the archive as well.

## Prepare a debugger for a remote debugging session

In a fist shell start `gdb`

```
$ gdb
      # Load an executable into the debugger session. gdb will read symbols and offsets from this
      # executable. This is needed to later be able to set breakpoints.
      # The executable here can either be init executable that was extracted in the optional
      # previous step or is an executable that is triggered by init and placed alongside in the
      # archive.
(gdb) file init
      # Breakpoints depend on the executable(s) that were loaded into the debugging session. As
      # init is a dynamically generated Go executable it will have the symbol main.main. The
      # function with this symbol will be executed first after the kernel finished its
      # initialization phase.
(gdb) break main.main
      # Instruct the debugger to connect to gdbserver.
(gdb) remote target localhost:1234
```
## Start the process that should be debugged

In a second shell run `qemu` with all the arguments as in [EXAMPLE.md](https://github.com/florianl/bluebox/blob/main/EXAMPLE.md) and add `-s` as argument. `-s` is shorthand for `-gdb tcp::1234` and will start a gdbserver on TCP port 1234 so the first shell in the previous step can connect to it.

```
$ qemu-system-x86_64 -s -nographic  -append "console=ttyS0" -m 4G -kernel /tmp/ci-kernels/linux-4.14.264.bz -initrd /tmp/initramfs.cpio
```
