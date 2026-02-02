import { listSessions, isUserActive } from '../lib/user.ts';
import chalk from 'chalk';
import { logger } from '../lib/logger.ts';

export async function listCommand() {
  try {
    const sessions = await listSessions();

    if (sessions.length === 0) {
      console.log(chalk.yellow('No active alcless sessions found.'));
      return;
    }

    console.log(chalk.bold(`📋 Active Sessions (${sessions.length}):`));
    console.log(chalk.gray('----------------------------------------'));
    
    for (const session of sessions) {
      const active = await isUserActive(session.username);
      const status = active ? chalk.green('● active') : chalk.red('○ inactive');
      console.log(`${chalk.cyan(session.instanceName.padEnd(20))} ${status} ${chalk.gray(`(${session.username})`)}`);
    }
    
    console.log(chalk.gray('----------------------------------------'));
  } catch (err: any) {
    logger.error(`Error listing sessions: ${err.message}`);
    process.exit(1);
  }
}
