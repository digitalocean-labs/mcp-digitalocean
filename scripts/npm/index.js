#!/usr/bin/env node

const path = require('path')
const fs = require('fs')
const childProcess = require('child_process')
const os = require('os')

/**
 * Run the platform-specific executable with the given arguments
 * @param {string[]} args Arguments to pass to the executable
 * @returns {Promise<number>} Returns a Promise that resolves to the exit code
 */
function runExecutable(args = []) {
    try {
        // Check for verbose flag
        const verbose = args.includes('--verbose')

        // Helper function to log only in verbose mode
        const verboseLog = (message) => {
            if (verbose) {
                console.error(message)
            }
        }

        const packageJson = require('./package.json')

        const platform = os.platform()
        const arch = os.arch()

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
        const execPath = path.join(__dirname, execName)
        verboseLog(`Executable path: ${execPath}`)

        if (!fs.existsSync(execPath)) {
            console.error(`Executable "${execPath}" not found.`)
            return Promise.resolve(1)
        }

        verboseLog(`Starting ${execPath}`)

        // Remove verbose flag before passing args to the child process
        const childArgs = args.filter(arg => arg !== '--verbose')

        const child = childProcess.spawn(execPath, childArgs, {
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

// Check if this file is being run directly
if (require.main === module) {
    // Run the executable with command line args and exit with its code
    runExecutable(process.argv.slice(2))
        .then(code => process.exit(code))
} else {
    // Export the function for consumers to use
    module.exports = { runExecutable }
}