use std::env;
use std::fs;
use std::path::Path;
use std::process::Command;

fn main() {
    let version = env::var("CARGO_PKG_VERSION").unwrap();
    let target_os = env::var("CARGO_CFG_TARGET_OS").unwrap();
    let target_arch = env::var("CARGO_CFG_TARGET_ARCH").unwrap();

    let platform = match target_os.as_str() {
        "macos" => "Darwin",
        "linux" => "Linux",
        "windows" => "Windows",
        _ => panic!("Unsupported OS: {}", target_os),
    };

    let arch = match target_arch.as_str() {
        "x86_64" => "x86_64",
        "aarch64" => "arm64",
        _ => panic!("Unsupported architecture: {}", target_arch),
    };

    let ext = if target_os == "windows" { "zip" } else { "tar.gz" };
    let filename = format!("omdr_{}_{}.{}", platform, arch, ext);
    let url = format!(
        "https://github.com/openmcpdirectory/omdr-cli/releases/download/v{}/{}",
        version, filename
    );

    let out_dir = env::var("OUT_DIR").unwrap();
    let download_path = Path::new(&out_dir).join(&filename);
    let bin_dir = Path::new(&out_dir).join("bin");

    fs::create_dir_all(&bin_dir).expect("Failed to create bin directory");

    println!("Downloading OMDR CLI from {}", url);

    let status = Command::new("curl")
        .args(&["-L", "-o", download_path.to_str().unwrap(), &url])
        .status()
        .expect("Failed to download binary");

    if !status.success() {
        panic!("Download failed");
    }

    if target_os == "windows" {
        Command::new("unzip")
            .args(&[download_path.to_str().unwrap(), "-d", bin_dir.to_str().unwrap()])
            .status()
            .expect("Failed to extract zip");
    } else {
        Command::new("tar")
            .args(&["-xzf", download_path.to_str().unwrap(), "-C", bin_dir.to_str().unwrap()])
            .status()
            .expect("Failed to extract tar.gz");
    }

    println!("cargo:rerun-if-changed=build.rs");
}
