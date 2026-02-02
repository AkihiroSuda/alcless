import { ensureSudo, run } from '../src/lib/exec.ts';
import { logger } from '../src/lib/logger.ts';
import chalk from 'chalk';

async function stage1() {
  console.log(chalk.bold.cyan('🛡️ Stage 1: Auth Guarantee\n'));

  try {
    logger.info('Verifying sudo access...');
    await ensureSudo();
    logger.success('Sudo is cached and active.');

    logger.info('Checking sysadminctl availability...');
    const check = await run('which', ['sysadminctl']);
    logger.success(`sysadminctl found at: ${check.stdout.trim()}`);

    console.log(chalk.bold.green('\n✅ Stage 1 Passed: Authentication is ready.'));
  } catch (err: any) {
    logger.error(`Stage 1 Failed: ${err.message}`);
    process.exit(1);
  }
}

stage1();
