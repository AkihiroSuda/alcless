import { getSessionUsername, isUserActive } from '../lib/user.ts';
import { logger } from '../lib/logger.ts';
import { ensureSudo } from '../lib/exec.ts';
import { spawn } from 'child_process';
import chalk from 'chalk';

/**
 * Executes a command in a session or drops into an interactive shell.
 */
export async function execCommand(instance: string, args: string[]) {
  try {
    const username = await getSessionUsername(instance);
    
    // Check if user is active
    if (!(await isUserActive(username))) {
      logger.error(`Instance "${instance}" is not active or does not exist. Run "alcless create ${instance}" first.`);
      process.exit(1);
    }

    // Ensure sudo is warmed up
    await ensureSudo();

    // Prepare the command for su
    // If no args, we just use su - username
    // If args, we use su - username -c "args..."
    const suArgs = ['-', username];
    if (args.length > 0) {
      suArgs.push('-c', args.join(' '));
    }

    // Use spawn with inherit for full interactivity (TTY support)
    const child = spawn('sudo', ['su', ...suArgs], {
      stdio: 'inherit',
      env: process.env,
    });

    child.on('exit', (code) => {
      process.exit(code || 0);
    });

    child.on('error', (err) => {
      logger.error(`Failed to start session: ${err.message}`);
      process.exit(1);
    });

  } catch (err: any) {
    logger.error(`Execution error: ${err.message}`);
    process.exit(1);
  }
}
