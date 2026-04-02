// Package compact provides lossless source-code compaction for reducing
// token usage when feeding code to AI models.
//
// It strips comments and removes blank lines from Go, Python, TypeScript,
// JavaScript, and Rust source files, producing valid, semantically-identical
// output. For Go files the standard AST parser is used so the result is
// always syntactically correct. For other languages a string-aware state
// machine correctly skips quoted literals before stripping comments.
package compact
