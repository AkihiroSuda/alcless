import { sudoers } from '../src/lib/sudo.ts';
import { getSessionUsername } from '../src/lib/user.ts';
import { run, runAsUser } from '../src/lib/exec.ts';
import { logger } from '../src/lib/logger.ts';
import chalk from 'chalk';
import fs from 'fs';

async function stage4() {
  const TEST_NAME = 'stage-test';
  const username = await getSessionUsername(TEST_NAME);

  console.log(chalk.bold.cyan('🔑 Stage 4: Access Guarantee\n'));

  try {
    logger.info(`Setting up sudoers for ${username}...`);
    await sudoers.setup(TEST_NAME);
    
    const filePath = sudoers.getFilePath(username);
    logger.info(`Verifying sudoers file at ${filePath}...`);
    
    if (fs.existsSync(filePath)) {
      logger.success(`Sudoers fragment found.`);
    } else {
      throw new Error(`Sudoers fragment NOT found at ${filePath}`);
    }

    logger.info('Validating sudoers syntax...');
    // Note: visudo -c usually requires root, but we've used sudo in setup
    const check = await run('sudo', ['visudo', '-c', '-f', filePath]);
    logger.success('Sudoers syntax is valid.');

    logger.info(`Testing no-password jump to ${username}...`);
    const whoami = await runAsUser(username, 'whoami');
    
    if (whoami.stdout.trim() === username) {
      logger.success(`Successfully entered session as ${username} without password.`);
    } else {
      throw new Error(`Jump failed: Expected ${username}, got ${whoami.stdout.trim()}`);
    }

    console.log(chalk.bold.green('\n✅ Stage 4 Passed: No-password access is guaranteed.'));
  } catch (err: any) {
    logger.error(`Stage 4 Failed: ${err.message}`);
    process.exit(1);
  }
}

stage4();
