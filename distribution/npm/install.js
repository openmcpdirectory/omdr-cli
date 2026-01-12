#!/usr/bin/env node

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const https = require('https');

const version = require('./package.json').version;
const platform = process.platform;
const arch = process.arch;

const platformMap = {
  darwin: 'Darwin',
  linux: 'Linux',
  win32: 'Windows'
};

const archMap = {
  x64: 'x86_64',
  arm64: 'arm64'
};

const ext = platform === 'win32' ? '.zip' : '.tar.gz';
const platformName = platformMap[platform];
const archName = archMap[arch];

if (!platformName || !archName) {
  console.error(`Unsupported platform: ${platform} ${arch}`);
  process.exit(1);
}

const filename = `omdr_${platformName}_${archName}${ext}`;
const url = `https://github.com/openmcpdirectory/omdr-cli/releases/download/v${version}/${filename}`;
const binDir = path.join(__dirname, 'bin');
const downloadPath = path.join(binDir, filename);

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

console.log(`Downloading OMDR CLI v${version} for ${platformName} ${archName}...`);

const file = fs.createWriteStream(downloadPath);

https.get(url, (response) => {
  if (response.statusCode === 302 || response.statusCode === 301) {
    https.get(response.headers.location, (redirectResponse) => {
      redirectResponse.pipe(file);
      file.on('finish', () => {
        file.close();
        extractAndCleanup();
      });
    });
  } else {
    response.pipe(file);
    file.on('finish', () => {
      file.close();
      extractAndCleanup();
    });
  }
}).on('error', (err) => {
  fs.unlinkSync(downloadPath);
  console.error(`Download failed: ${err.message}`);
  process.exit(1);
});

function extractAndCleanup() {
  console.log('Extracting binary...');
  
  try {
    if (platform === 'win32') {
      execSync(`tar -xf "${downloadPath}" -C "${binDir}"`, { stdio: 'inherit' });
    } else {
      execSync(`tar -xzf "${downloadPath}" -C "${binDir}"`, { stdio: 'inherit' });
    }
    
    fs.unlinkSync(downloadPath);
    
    const binaryName = platform === 'win32' ? 'omdr.exe' : 'omdr';
    const binaryPath = path.join(binDir, binaryName);
    
    if (platform !== 'win32') {
      fs.chmodSync(binaryPath, 0o755);
    }
    
    console.log('OMDR CLI installed successfully!');
    console.log(`Run 'omdr --help' to get started.`);
  } catch (err) {
    console.error(`Extraction failed: ${err.message}`);
    process.exit(1);
  }
}
