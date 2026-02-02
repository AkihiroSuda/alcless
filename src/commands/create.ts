import { createSessionUser } from '../lib/user.ts';
import { sudoers } from '../lib/sudo.ts';
import { provisionSession } from '../lib/provision.ts';
import { MultiBar, Presets } from 'cli-progress';
import pLimit from 'p-limit';
import chalk from 'chalk';
import { logger } from '../lib/logger.ts';
import { ensureSudo } from '../lib/exec.ts';

export async function createCommand(instances: string[], options: { tools?: string; concurrency?: string }) {
  if (instances.length === 0) {
    logger.error('Please specify at least one instance name.');
    process.exit(1);
  }

  if (options.tools) {
    logger.warn('The --tools flag is currently disabled as Homebrew support has been removed.');
  }

  // Pre-flight sudo check
  await ensureSudo();

  console.log(chalk.bold.cyan('💡 Pro Tip: ') + chalk.white('To avoid macOS permission popups, grant Ghostty ') + chalk.bold.yellow('"Full Disk Access"') + chalk.white(' in System Settings.\n'));
  
  const concurrency = parseInt(options.concurrency || '2', 10);
  const limit = pLimit(concurrency);

  console.log(chalk.bold(`🚀 Creating ${instances.length} session(s) (concurrency: ${concurrency})...`));

  const multibar = new MultiBar({
    clearOnComplete: false,
    hideCursor: true,
    format: '{bar} | {percentage}% | {instance} | {step}',
  }, Presets.shades_grey);

  const tasks = instances.map(instance => limit(async () => {
    const bar = multibar.create(100, 0, { instance, step: 'Initializing...' });
    
    try {
      // Step 1: Create User
      bar.update(10, { step: 'Creating user...' });
      await createSessionUser(instance);
      
      // Step 2: Sudoers
      bar.update(30, { step: 'Configuring sudo...' });
      await sudoers.setup(instance);
      
      // Step 3: Provision tools
      bar.update(50, { step: 'Provisioning tools (gh, jq, uv, python, bun)...' });
      await provisionSession(instance);
      
      bar.update(100, { step: 'Finished!' });
    } catch (err: any) {
      bar.update(0, { step: chalk.red('Failed!') });
      multibar.stop();
      logger.error(`Error creating instance "${instance}": ${err.message}`);
      process.exit(1);
    }
  }));

  await Promise.all(tasks);
  multibar.stop();
  console.log(chalk.bold.green('\n✅ All sessions created successfully.'));
}
