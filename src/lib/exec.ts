import { execa, type Options, type Result } from 'execa';
import { logger } from './logger.ts';
import fs from 'fs';
import path from 'path';

const LOG_DIR = '.alcless/logs';
const TRACE_LOG = path.join(LOG_DIR, 'trace.log');

// Ensure log directory exists
if (!fs.existsSync(LOG_DIR)) {
  fs.mkdirSync(LOG_DIR, { recursive: true });
}

export interface ExecOptions extends Options {
  /**
   * If true, suppresses stdout and stderr in the CLI.
   * All output is still sent to the trace log.
   */
  silent?: boolean;
  /**
   * Optional timeout in milliseconds.
   */
  timeoutMs?: number;
}

export interface RunResult {
  stdout: string;
  stderr: string;
  exitCode: number;
  command: string;
}

/**
 * Appends a message to the trace log file.
 */
function trace(message: string) {
  const timestamp = new Date().toISOString();
  fs.appendFileSync(TRACE_LOG, `[${timestamp}] ${message}\n`);
}

/**
 * Ensures sudo is authenticated before starting automated tasks.
 */
export async function ensureSudo(): Promise<void> {
  try {
    trace('CMD: sudo -v');
    await execa('sudo', ['-v'], { stdio: 'inherit' });
  } catch (err) {
    trace(`AUTH FAILED: ${err}`);
    throw new Error('Sudo authentication failed. This tool requires sudo privileges.');
  }
}

/**
 * Executes a command with focused logging and trace capturing.
 */
export async function run(
  file: string,
  args: string[],
  options: ExecOptions = {}
): Promise<RunResult> {
  const { silent = true, timeoutMs = 120000, ...execaOptions } = options;

  const commandStr = `${file} ${args.join(' ')}`;
  trace(`EXEC: ${commandStr}`);

  const subprocess = execa(file, args, {
    ...execaOptions,
    timeout: timeoutMs,
    all: true,
  });

  // Heartbeat/Watchdog
  let elapsed = 0;
  const heartbeat = setInterval(() => {
    elapsed += 5;
    if (file === 'sudo' && args.includes('sysadminctl')) {
      logger.info(`[PID: ${subprocess.pid}] Waiting for macOS administrative permission... (${elapsed}s)`);
      trace(`STILL RUNNING: ${commandStr} (PID: ${subprocess.pid}, ELAPSED: ${elapsed}s)`);
    }
  }, 5000);

  try {
    const result = await (subprocess as any);
    
    // Log complete output to trace
    if (result.all) {
      trace(`RESULT [${result.exitCode}]:\n${result.all}`);
    }

    return {
      stdout: String(result.stdout ?? ''),
      stderr: String(result.stderr ?? ''),
      exitCode: result.exitCode ?? 0,
      command: result.command ?? '',
    };
  } catch (error: any) {
    const result = error as Result;
    if (error.timedOut) {
      trace(`TIMEOUT: ${commandStr} after ${timeoutMs}ms`);
      throw new Error(`Command timed out after ${timeoutMs}ms: ${commandStr}. 
HINT: This usually happens when macOS is waiting for a GUI permission click. 
Check your screen for popups or grant your Terminal "Full Disk Access" in System Settings.`);
    }
    
    trace(`FAILED [${result.exitCode}]: ${result.all || result.message}`);
    throw new Error(
      `Command failed: ${result.command}\n` +
      `Exit code: ${result.exitCode}\n` +
      `Output: ${result.all || result.stderr || result.message}`
    );
  } finally {
    clearInterval(heartbeat);
  }
}

/**
 * Executes a command with sudo.
 */
export async function sudoRun(
  file: string,
  args: string[],
  options: ExecOptions = {}
): Promise<RunResult> {
  const useInteractive = options.stdio === 'inherit';
  const sudoArgs = useInteractive ? [file, ...args] : ['-n', file, ...args];
  return run('sudo', sudoArgs, options);
}

/**
 * Executes a command as a specific user using sudo su -.
 */
export async function runAsUser(
  username: string,
  command: string,
  options: ExecOptions = {}
): Promise<RunResult> {
  return sudoRun('su', ['-', username, '-c', command], options);
}
