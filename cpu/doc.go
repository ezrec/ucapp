// Package cpu implements the microprocessor and assembler for the μCAPP system.
//
// The CPU consists of an instruction pointer (IP) with three execution modes,
// six 32-bit general-purpose registers (r0-r5), an ALU, a stack, and conditional
// execution flags. The processor coordinates with the Content Addressable Parallel
// Processor (CAPP) through Match/Mask registers.
//
// The assembler provides a custom assembly language for the μCAPP instruction set,
// supporting macros, labels, equates, and compile-time expression evaluation.
package cpu
