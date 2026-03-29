//! NeoHtop Core - Rust monitoring library (FFI backend)
//!
//! NOTE: This library is not currently integrated into the NeoHtopCLI build.
//! The CLI uses pure Go monitoring (cli/monitor/) instead.
//! Kept for potential future use or as reference for the NeoHtop desktop app.

pub mod ffi;
pub mod monitoring;
pub mod state;
