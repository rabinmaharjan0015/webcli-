#!/usr/bin/env node
const { execSync } = require('child_process');
const { existsSync, mkdirSync, createWriteStream, chmodSync } = require('fs');
const { get } = require('https');
const { platform, arch, homedir } = require('os');
const { join } = require('path');
const { createGunzip } = require('zlib');

const REPO = 'yenya/webcli';
const BIN_DIR = join(__dirname, 'bin');
const BIN_PATH = join(BIN_DIR, 'webcli' + (process.platform === 'win32' ? '.exe' : ''));

function getPlatform() {
  const osMap = { darwin: 'darwin', linux: 'linux', win32: 'windows' };
  const archMap = { x64: 'amd64', arm64: 'arm64' };
  const os = osMap[platform()];
  const a = archMap[arch()];
  if (!os) throw new Error(`Unsupported OS: ${platform()}`);
  if (!a) throw new Error(`Unsupported arch: ${arch()}`);
  return `${os}_${a}`;
}

function getLatestVersion() {
  return new Promise((resolve, reject) => {
    get(`https://api.github.com/repos/${REPO}/releases/latest`, {
      headers: { 'User-Agent': 'webcli-installer', Accept: 'application/json' },
    }, (res) => {
      let data = '';
      res.on('data', (c) => data += c);
      res.on('end', () => {
        try {
          const tag = JSON.parse(data).tag_name;
          resolve(tag);
        } catch {
          reject(new Error('Failed to parse latest version'));
        }
      });
    }).on('error', reject);
  });
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = createWriteStream(dest);
    get(url, { headers: { 'User-Agent': 'webcli-installer' } }, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        file.close();
        return download(res.headers.location, dest).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) {
        file.close();
        return reject(new Error(`HTTP ${res.statusCode}`));
      }
      res.pipe(createGunzip()).pipe(file);
      file.on('finish', () => file.close(resolve));
    }).on('error', (e) => { file.close(); reject(e); });
  });
}

async function install() {
  // Check if binary already exists
  if (existsSync(BIN_PATH)) {
    console.log('webcli: binary already installed');
    process.exit(0);
  }

  if (!existsSync(BIN_DIR)) {
    mkdirSync(BIN_DIR, { recursive: true });
  }

  try {
    const platform = getPlatform();
    const version = await getLatestVersion();
    const url = `https://github.com/${REPO}/releases/download/${version}/webcli_${platform}.tar.gz`;

    console.log(`webcli: downloading ${version} (${platform})...`);
    await download(url, BIN_PATH);
    chmodSync(BIN_PATH, 0o755);
    console.log(`webcli: installed to ${BIN_PATH}`);

    // Create symlink for direct access
    const homeBin = join(homedir(), '.local', 'bin');
    if (existsSync(homeBin)) {
      try {
        execSync(`ln -sf "${BIN_PATH}" "${join(homeBin, 'webcli')}"`);
      } catch {}
    }
  } catch (e) {
    // Fallback: try building from source
    console.log('webcli: binary download failed, trying go install...');
    try {
      execSync(`go install github.com/${REPO}@latest`, { stdio: 'inherit' });
      const goBin = execSync('go env GOPATH').toString().trim() + '/bin/webcli';
      if (existsSync(goBin)) {
        execSync(`cp "${goBin}" "${BIN_PATH}"`);
        console.log(`webcli: installed to ${BIN_PATH}`);
      }
    } catch (e2) {
      console.error('webcli: install failed. Run: go install github.com/${REPO}@latest');
      console.error(e.message);
      process.exit(1);
    }
  }
}

install();
