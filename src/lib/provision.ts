import { runAsUser } from './exec.ts';
import { getSessionUsername } from './user.ts';

/**
 * Provisions the session with the default toolchain.
 */
export async function provisionSession(instanceName: string): Promise<void> {
  const sessionUser = await getSessionUsername(instanceName);
  
  // Install tools via curl scripts
  const installCmd = [
    'curl -sS https://webi.sh/gh | sh',
    'curl -sS https://webi.sh/jq | sh',
    'source ~/.config/envman/PATH.env',
    'curl -LsSf https://astral.sh/uv/install.sh | sh',
    '$HOME/.local/bin/uv python install 3.12',
    'curl -fsSL https://bun.com/install | bash'
  ].join(' && ');

  await runAsUser(sessionUser, `bash -c '${installCmd}'`);

  // Ensure tool paths are added to shell profiles for future interactive sessions
  const profileCmds = [
    'grep -q "envman/PATH.env" ~/.zprofile || echo "source ~/.config/envman/PATH.env" >> ~/.zprofile',
    'grep -q "envman/PATH.env" ~/.bash_profile || echo "source ~/.config/envman/PATH.env" >> ~/.bash_profile',
    'grep -q ".local/bin" ~/.zprofile || echo "export PATH=\\"$HOME/.local/bin:\\$PATH\\"" >> ~/.zprofile',
    'grep -q ".local/bin" ~/.bash_profile || echo "export PATH=\\"$HOME/.local/bin:\\$PATH\\"" >> ~/.bash_profile',
    'grep -q ".bun/bin" ~/.zprofile || echo "export PATH=\\"$HOME/.bun/bin:\\$PATH\\"" >> ~/.zprofile',
    'grep -q ".bun/bin" ~/.bash_profile || echo "export PATH=\\"$HOME/.bun/bin:\\$PATH\\"" >> ~/.bash_profile'
  ].join(' && ');

  await runAsUser(sessionUser, `bash -c '${profileCmds}'`);
}
