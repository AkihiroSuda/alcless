import { isUserActive, getSessionUsername } from '../src/lib/user.ts';
import { logger } from '../src/lib/logger.ts';
import { sudoRun } from '../src/lib/exec.ts';
import chalk from 'chalk';

async function stage3() {
  const TEST_NAME = 'stage-test';
  const username = await getSessionUsername(TEST_NAME);

  console.log(chalk.bold.cyan('🔄 Stage 3: Propagation Guarantee\n'));

  try {
    logger.info(`Checking if ${username} is recognized by the Unix layer (id)...`);
    
    let active = false;
    for (let i = 0; i < 5; i++) {
      if (await isUserActive(username)) {
        active = true;
        break;
      }
      logger.warn(`User ${username} not recognized yet, flushing cache... (Attempt ${i+1}/5)`);
      await sudoRun('dscacheutil', ['-flushcache']);
      await new Promise(resolve => setTimeout(resolve, 2000));
    }

    if (active) {
      logger.success(`${username} is live and recognized by the system.`);
    } else {
      throw new Error(`Propagation Timeout: ${username} exists in dscl but is NOT recognized by the 'id' command.`);
    }

    console.log(chalk.bold.green('\n✅ Stage 3 Passed: Identity has propagated to the system.'));
  } catch (err: any) {
    logger.error(`Stage 3 Failed: ${err.message}`);
    process.exit(1);
  }
}

stage3();
