use std::env;
use std::process::{Command, exit};

fn main() {
    let binary_name = if cfg!(windows) { "omdr.exe" } else { "omdr" };
    let binary_path = env::current_exe()
        .expect("Failed to get current executable path")
        .parent()
        .expect("Failed to get parent directory")
        .join(binary_name);

    let args: Vec<String> = env::args().skip(1).collect();

    let status = Command::new(binary_path)
        .args(&args)
        .status()
        .expect("Failed to execute omdr binary");

    exit(status.code().unwrap_or(1));
}
