import { createSessionUser, userExists, isUserActive } from '../src/lib/user.ts';
import { logger } from '../src/lib/logger.ts';
import chalk from 'chalk';

async function stage2() {
  const TEST_NAME = 'stage-test';
  const username = `alcl_${process.env.USER}_${TEST_NAME}`;

  console.log(chalk.bold.cyan('👤 Stage 2: Record Guarantee\n'));

  try {
    logger.info(`Attempting to create user: ${username}...`);
    
    // We run the creation logic
    const createdUser = await createSessionUser(TEST_NAME);

    logger.info('Verifying record in Directory Service (dscl)...');
    const exists = await userExists(createdUser);
    
    if (exists) {
      logger.success(`Directory Service record found for ${createdUser}.`);
    } else {
      throw new Error(`User creation reported success but record not found in dscl for ${createdUser}.`);
    }

    console.log(chalk.bold.green('\n✅ Stage 2 Passed: User record created and verified.'));
  } catch (err: any) {
    logger.error(`Stage 2 Failed: ${err.message}`);
    process.exit(1);
  }
}

stage2();
