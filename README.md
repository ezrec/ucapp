# μCAPP - (micro) Content Adressable Parallel Processor

The μCAPP is an abstract processor, based on concepts from Caxton C. Foster's
1976 book "Content Addressable Parallel Processors" (ISBN 0-442-22433-8).

## Getting Started

### Running a ucapp program.

Providing a .uc file to ucapp will assemble then file, clear the CAPP, load
the assembled file into the CAPP, reset the instruction pointer, and execute
the program.

```
$ go run ./cmd/ucapp -c examples/io/hello_world.uc
Hello World!
```

## Historical Perspective

μCAPP is an unusual system from the modern view - memory is not addressable by
index (ie, `mem_array[N] has value X`), but by content (ie, `return `_all_`
memory elements where value is similar to X`).

However, even in modern systems, a content addressable memory can be found in
specific applications - for example, the TCAM memories used by network routers
to look up policies based on IP or MAC addresses.

But professor Foster expanded on that concept, and designed computer
architectures where the content addressable memory was not simply a fast-lookup
peripheral, but the actual main memory of the system _and_ a unique bit-mask
vector processor.

He later worked on the design of the Goodyear STARAN system (1977) - developed
for the US Air Force for air traffic control situations.

μCAPP looks to imagine a retro-history where STARAN technology was made readily
avaible to the interested researcher - as the Intel 8080 and MOS Technology
6502 brought conventional linear memory array processors to the hobbyists of
the time.

## Architecture

The μCAPP is composed of the following components:

- Content Addressable Parallel Processor (main memory)
- Microprocessor
  - Instruction pointer
  - Six 32-bit registers
  - ALU
  - Conditional execution flags
  - Stack
- I/O channels

### CLI Interface

See [cmd/ucapp/README.md](cmd/ucapp/README.md)

### CAPP Details

See [capp/README.md](capp/README.md)

### Microprocessor

See [cpu/README.md](cpu/README.md)

### I/O Peripherals

See [io/README.md](io/README.md)
