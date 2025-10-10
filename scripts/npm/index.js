#!/usr/bin/env node

import { fileURLToPath } from 'url'
import { dirname, join } from 'path'
import { existsSync } from 'fs'
import { spawn } from 'child_process'
import { platform as _platform, arch as _arch } from 'os'

/**
 * Run the platform-specific executable with the given arguments
 * @param {string[]} args Arguments to pass to the executable
 * @returns {Promise<number>} Returns a Promise that resolves to the exit code
 */
export async function runExecutable(args = []) {
    try {
        // Check for verbose flag
        const verbose = args.includes('--verbose')

        // Helper function to log only in verbose mode
        const verboseLog = (message) => {
            if (verbose) {
                console.error(message)
            }
        }

        const { default: packageJson } = await import('./package.json', { assert: { type: 'json' } });

        const platform = _platform()
        const arch = _arch()

        verboseLog(`Detected platform: ${platform}`)
        verboseLog(`Detected architecture: ${arch}`)

        const binKey = `mcp-digitalocean-${platform}-${arch}`;
        const execName = packageJson["mcp-server-binaries"][binKey]

        // Some error messages should always show regardless of verbose mode
        if (!execName) {
            console.error(`No executable found for platform: ${platform}-${arch}`)
            return Promise.resolve(1)
        }

        verboseLog(`Found executable in package.json: ${execName}`)

        // The platform-specific executable should be in the same folder
        const execPath = join(dirname(fileURLToPath(import.meta.url)), execName);
        verboseLog(`Executable path: ${execPath}`)

        if (!existsSync(execPath)) {
            console.error(`Executable "${execPath}" not found.`)
            return Promise.resolve(1)
        }

        verboseLog(`Starting ${execPath}`)

        // Remove verbose flag before passing args to the child process
        const childArgs = args.filter(arg => arg !== '--verbose')

        const child = spawn(execPath, childArgs, {
            stdio: 'inherit',
            shell: false
        })

        return new Promise((resolve) => {
            child.on('error', (err) => {
                console.error(`Error executing package: ${err.message}`)
                resolve(1)
            })

            child.on('exit', (code) => {
                resolve(code || 0)
            })
        })
    } catch (err) {
        console.error(`Error running executable: ${err.message}`)
        return Promise.resolve(1)
    }
}

runExecutable(process.argv.slice(2))
    .then(code => process.exit(code))