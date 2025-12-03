package internal

import (
	"iter"
)

// IterSeqConcat concatenates multiple iterators into a single iterator sequence.
func IterSeqConcat[T any](seqs ...iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, seq := range seqs {
			for val := range seq {
				if !yield(val) {
					return // Stop if the consumer stops
				}
			}
		}
	}
}

// IterSeq2Concat concatenates multiple dual-return iterators into a single iterator sequence.
func IterSeq2Concat[T1 any, T2 any](seqs ...iter.Seq2[T1, T2]) iter.Seq2[T1, T2] {
	return func(yield func(T1, T2) bool) {
		for _, seq := range seqs {
			for val1, val2 := range seq {
				if !yield(val1, val2) {
					return // Stop if the consumer stops
				}
			}
		}
	}
}
