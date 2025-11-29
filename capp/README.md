# Content Addressable Parallel Processor

The CAPP is based on concepts from Caxton C. Foster's 1976 book
"Content Addressable Parallel Processors" (ISBN 0-442-22433-8).

## Overview

A Content Addressable Parallel Processor (CAPP) is a memory system fundamentally different from traditional indexed memory. Instead of accessing data by specifying an address or index, CAPP allows you to access data by specifying its content. This enables powerful parallel operations where all cells matching certain criteria can be operated upon simultaneously.

## Key Concepts

### Content-Addressable Memory

In traditional memory systems, you might say "give me the data at address 0x1000." In CAPP, you say "give me all cells whose data matches this pattern." This shift from location-based to content-based addressing enables entirely different programming paradigms, particularly suited for:

- Database-like searches and queries
- Pattern matching across large datasets
- Parallel data transformations
- Set-based operations
- Associative lookups

### Parallel Processing

CAPP operations work on all matching cells simultaneously. When you issue a command like "add 1 to all cells containing even numbers," the CAPP processes all qualifying cells in parallel, making certain operations dramatically faster than sequential processing.

## Memory Cell Structure

Each CAPP memory cell contains:

- **Set bits (2)**: Two selection bits that determine which cells participate in operations. The system can swap between two independent sets, allowing complex multi-stage operations.
- **Tag bit**: Marks cells for inclusion in the current operation list
- **Data (32-bit)**: The actual data value stored in the cell

## The Three-Level Selection Hierarchy

CAPP uses a three-level hierarchy to select which cells participate in operations:

### 1. Set Selection (Set Bits)

The Set bits provide two independent selection states. At any time, one set is active. You can:
- Select cells matching a content pattern into the active set using `SET_OF`
- Swap between the two sets using `SET_SWAP`

This dual-set system is used internally by the CPU to preserve list state between instruction
execution and instruction decode phases.

### 2. Tagging (Tag Bit)

Within the active set, the Tag bit marks which cells are in the "active list." Operations like:
- `LIST_ALL` - Tag all cells in the active set.
- `LIST_ONLY` - Refine the tag list by content matching.
- `LIST_NOT` - Invert which cells are tagged.
- `LIST_NEXT` - Clear the Tag bit of the first list cell, removing it from the active list.

### 3. Operation Target

Some operations target different subsets of tagged cells:
- `WRITE_LIST` - Modify all tagged cells
- `WRITE_FIRST` - Modify only the first tagged cell

### 4. Properties

- `First()` - Read only the first cell in the active list.
- `Count()` - Number of cells in the active list.
- `List()` - Iterator over all cells in the active list, in order.
- `BitsFlipped` - Count of all bit changes since last reset (useful for energy/performance modeling).

## Operations

CAPP provides eight fundamental operations:

### Selection Operations

**`SET_OF(match, mask)`**: Select cells into the active set where `(cell.Data & mask) == (match & mask)`. This is the primary way to select cells by content.

**`SET_SWAP`**: Swap between the two independent selection sets, enabling complex multi-stage queries.

### Tagging Operations

**`LIST_ALL`**: Tag all cells in the active set for processing.

**`LIST_ONLY(match, mask)`**: Refine the tagged list, keeping only cells where `(cell.Data & mask) == (match & mask)`.

**`LIST_NOT`**: Invert the tag bits for all cells in the active set.

**`LIST_NEXT`**: Remove the first tagged cell from the list, advancing to the next cell.

### Data Modification Operations

**`WRITE_LIST(value, mask)`**: Write to all tagged cells. Bits set in `mask` are replaced with corresponding bits from `value`: `cell.Data = (cell.Data & ^mask) | (value & mask)`.

**`WRITE_FIRST(value, mask)`**: Like `WRITE_LIST`, but only modifies the first tagged cell.

## Match/Mask Pattern

Many CAPP operations use a match/mask pattern:
- The **mask** specifies which bits to examine
- The **match** specifies what value those bits should have

For example:
- `match=0x00FF`, `mask=0x00FF` - Match cells where the low byte is 0xFF
- `match=0x0000`, `mask=0x8000` - Match cells where bit 15 is 0
- `match=0xFFFF`, `mask=0xFFFF` - Match cells equal to 0xFFFF

This allows precise content-based selection without needing separate comparison operations.

## Observability

The CAPP tracks several metrics:

- **`Count()`**: Number of currently tagged cells
- **`First()`**: Data value of the first tagged cell
- **`List()`**: Iterator over all tagged cells in order
- **`BitsFlipped`**: Count of all bit changes since last reset (useful for energy/performance modeling)

## Example Usage Pattern

```go
// Select all even cells with values >= 0x8000 (high bit set)
capp.Action(SET_OF, 0x8000, 0x8001)

// Tag them all for processing
capp.Action(LIST_ALL, 0, 0)

// Add 1 to all tagged cells (simulated with mask operations) by setting the low bit.
capp.Action(WRITE_LIST, 1, 1)
```

## Historical Context

CAPP architectures emerged in the 1970s with systems like STARAN, developed for the US Air Force for radar signal processing and database operations. These systems could perform parallel searches across thousands of data elements simultaneously, making them ideal for real-time pattern matching and associative retrieval tasks that would bog down conventional processors.

The μCAPP implementation explores what a hobbyist-accessible version of this technology might have looked like had it followed a similar trajectory to the microprocessor revolution.
