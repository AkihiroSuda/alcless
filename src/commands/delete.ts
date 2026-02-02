import { deleteSessionUser } from '../lib/user.ts';
import { sudoers } from '../lib/sudo.ts';
import { MultiBar, Presets } from 'cli-progress';
import pLimit from 'p-limit';
import chalk from 'chalk';
import { logger } from '../lib/logger.ts';
import { ensureSudo } from '../lib/exec.ts';

export async function deleteCommand(instances: string[], options: { concurrency?: string }) {
  if (instances.length === 0) {
    logger.error('Please specify at least one instance name.');
    process.exit(1);
  }

  // Pre-flight sudo check
  await ensureSudo();

  const concurrency = parseInt(options.concurrency || '4', 10);
  const limit = pLimit(concurrency);

  console.log(chalk.bold(`🗑️ Deleting ${instances.length} session(s)...`));

  const multibar = new MultiBar({
    clearOnComplete: false,
    hideCursor: true,
    format: '{bar} | {percentage}% | {instance} | {step}',
  }, Presets.shades_grey);

  const tasks = instances.map(instance => limit(async () => {
    const bar = multibar.create(100, 0, { instance, step: 'Deleting...' });
    
    try {
      bar.update(30, { step: 'Removing sudoers...' });
      await sudoers.remove(instance);
      
      bar.update(60, { step: 'Deleting user...' });
      await deleteSessionUser(instance);
      
      bar.update(100, { step: 'Done' });
    } catch (err: any) {
      bar.update(0, { step: chalk.red('Failed!') });
      multibar.stop();
      logger.error(`Error deleting instance "${instance}": ${err.message}`);
    }
  }));

  await Promise.all(tasks);
  multibar.stop();
  console.log(chalk.bold.green('\n✅ Sessions deleted.'));
}
