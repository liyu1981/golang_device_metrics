const { execSync } = require('child_process');
const { mkdirSync, existsSync } = require('fs');
const { platform, arch } = require('os');
const path = require('path');

const BIN_DIR = path.join(__dirname, '../bin');
mkdirSync(BIN_DIR, { recursive: true });

/**
 * @param {string} tool 
 * @param {string} version 
 */
function getDownloadUrl(tool, version) {
    const plat = platform();
    const archMap = {
        x64: 'amd64',
        arm64: 'arm64'
    };
    const archSuffix = archMap[arch()] || 'amd64';

    if (tool === 'golangci-lint') {
        // refer to https://github.com/golangci/golangci-lint/releases, for downloading link patterns
        return `https://github.com/golangci/golangci-lint/releases/download/v${version}/golangci-lint-${version}-` +
            `${plat === 'darwin' ? 'darwin' : 'linux'}-${archSuffix}.tar.gz`;
    }

    if (tool === 'typos') {
        const ext = plat === 'win32' ? '.exe' : '';
        // refer to https://github.com/crate-ci/typos/releases, for downloading link patterns
        return `https://github.com/crate-ci/typos/releases/download/v${version}/typos-v${version}-${archSuffix === 'amd64' ? 'x86_64' : 'aarch64'}-${plat === 'darwin' ? 'apple-darwin' : 'unknown-linux-musl'}.tar.gz`;
    }

    if (tool === 'air') {
        const ext = plat === 'win32' ? '.exe' : '';
        const os = plat === 'darwin' ? 'darwin' : 'linux';
        // refer to https://github.com/cosmtrek/air/releases, for downloading link patterns
        return `https://github.com/cosmtrek/air/releases/download/v${version}/air_${version}_${os}_${archSuffix}${ext}`;
    }

    throw new Error(`Unknown tool: ${tool}`);
}

function verifyTool(binPath) {
    const name = path.basename(binPath);
    const cmdMap = {
        'golangci-lint': '--version',
        'typos': '--version',
        'air': '-v'
    };

    const flag = cmdMap[name] || '--version';

    try {
        const output = execSync(`${binPath} ${flag}`, { encoding: 'utf-8' });
        console.log(`✅ ${name} verified`);
    } catch (err) {
        console.error(`❌ ${name} verification failed:`, err.message);
        throw err;
    }
}

/**
 * @param {string} version version without leading 'v', like '2.2.1'
 */
function installGolangciLint(version) {
    const url = getDownloadUrl('golangci-lint', version);
    console.log(`Downloading golangci-lint from ${url}`);
    execSync(`curl -sSL ${url} | tar xz -C ${BIN_DIR} --strip-components=1`, { stdio: 'inherit' });
    verifyTool('./bin/golangci-lint');
}

/** 
 * @param version {string} version without leading 'v', like '1.34.0'
*/
function installTypos(version) {
    const url = getDownloadUrl('typos', version);
    console.log(`Downloading typos from ${url}`);
    execSync(`curl -sSL ${url} | tar xz -C ${BIN_DIR} --strip-components=1`, { stdio: 'inherit' });
    verifyTool('./bin/typos');
}

/**
 * @param {string} version version without leading 'v', like '1.62.0'
 */
function installAir(version) {
    const url = getDownloadUrl('air', version);
    const ext = platform() === 'win32' ? '.exe' : '';
    const dest = path.join(BIN_DIR, `air${ext}`);
    console.log(`Downloading air from ${url}`);
    execSync(`curl -sSL -o ${dest} ${url}`, { stdio: 'inherit' });
    execSync(`chmod +x ${dest}`);
    verifyTool('./bin/air');
}

try {
    installGolangciLint('2.2.1');
    installTypos('1.34.0');
    installAir('1.62.0');
    console.log('✅ Tools installed in ./bin');
} catch (err) {
    console.error('❌ Failed to install tools:', err.message);
    process.exit(1);
}
