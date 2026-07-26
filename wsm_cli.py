#!/usr/bin/env python3
"""WSM Workspace Manager — CLI front-end."""

import argparse
import itertools
import os
import subprocess
import sys
import time
from pathlib import Path

from wsm_core import CONF_DIR, VERSION, parse_toml, is_mounted, check_net

SPINNER = itertools.cycle('⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏')
IS_TTY = sys.stdout.isatty()


def spin(msg):
    """Print spinning progress indicator on current line."""
    if IS_TTY:
        print(f'\r{msg} {next(SPINNER)}', end='', flush=True)
    else:
        print(f'{msg}...')


def spin_done(msg='OK'):
    """Clear spinner line and print completion."""
    if IS_TTY:
        try:
            tw = os.get_terminal_size().columns
        except OSError:
            tw = 80
        print(f'\r{" " * tw}\r{msg}', flush=True)
    else:
        print(msg)


def cmd_mount(remote_path, local_mount):
    """Mount remote path via SSHFS."""
    alias = remote_path.split(':')[0]
    spin(f'Resolving {alias}')
    ok, err = check_net(alias)
    if not ok:
        spin_done(f'FAILED: {err}')
        sys.exit(1)
    spin_done(f'Resolved {alias}')

    Path(local_mount).mkdir(parents=True, exist_ok=True)
    proc = subprocess.Popen([
        'sshfs', remote_path, local_mount,
        '-o', 'cache=yes,cache_stat_timeout=1200,cache_dir_timeout=1200,'
              'cache_link_timeout=1200',
        '-o', 'compression=yes,reconnect,ServerAliveInterval=15',
        '-o', 'kernel_cache,noauto_cache',
    ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    while proc.poll() is None:
        spin(f'Mounting {alias} -> {local_mount}')
        time.sleep(0.1)
    if proc.returncode == 0:
        spin_done(f'Mounted {alias}')
    else:
        spin_done(f'Mount FAILED (code {proc.returncode})')


def cmd_unmount(local_mount):
    """Safely unmount with fuser cleanup and lazy-unmount fallback."""
    if not is_mounted(local_mount):
        print('Project directory is not mounted.')
        return

    print('Terminating locking file descriptors via fuser...')
    subprocess.run(
        ['fuser', '-k', '-M', local_mount],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )

    print(f'Unmounting {local_mount}...')
    result = subprocess.run(
        ['umount', local_mount], capture_output=True,
    )
    if result.returncode != 0:
        print('Standard unmount blocked. Applying safe lazy-unmount...')
        subprocess.run(['umount', '-l', local_mount])

    print('Workspace detached successfully.')


def cmd_connect(remote_path, editor_cmd):
    """Launch editor native SSH remoting (bypasses FUSE)."""
    ssh_alias = remote_path.split(':')[0]
    remote_dir = remote_path[len(ssh_alias):]

    spin(f'Probing {ssh_alias}')
    ok, err = check_net(ssh_alias)
    if not ok:
        spin_done(f'FAILED: {err}')
        sys.exit(1)
    spin_done(f'Reachable: {ssh_alias}')

    if editor_cmd == 'code':
        uri = f'vscode-remote://ssh-remote+{ssh_alias}{remote_dir}'
        print(f'Launching VS Code Remote-SSH on {ssh_alias}...')
        subprocess.Popen(
            ['code', '--folder-uri', uri],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        )
    else:
        uri = f'ssh://{ssh_alias}{remote_dir}'
        print(f'Launching {editor_cmd} native SSH Server on {ssh_alias}...')
        subprocess.Popen(
            [editor_cmd, uri],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        )


def cmd_run(remote_path, local_mount, editor_cmd, _project):
    """Mount (if needed) and launch editor."""
    if not is_mounted(local_mount):
        cmd_mount(remote_path, local_mount)

    timeout = 50
    while not is_mounted(local_mount):
        spin(f'VFS wait {local_mount}')
        time.sleep(0.1)
        timeout -= 1
        if timeout <= 0:
            spin_done('VFS TIMEOUT')
            sys.exit(1)
    spin_done('VFS ready')

    print(f'Launching {editor_cmd}...')
    subprocess.Popen(
        [editor_cmd, local_mount],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )


def list_projects():
    """List available project names."""
    if not CONF_DIR.exists():
        return []
    return sorted(
        f.stem for f in CONF_DIR.glob('*.conf') if f.is_file()
    )


def cmd_delete(project, conf, local_mount, force=False):
    """Delete project config with confirmation and safety checks."""
    if local_mount and is_mounted(local_mount):
        print(f'Error: {project} is currently mounted. Unmount first.',
              file=sys.stderr)
        sys.exit(1)

    if not force:
        ans = input(f'Delete config "{project}"? [y/N]: ').strip().lower()
        if ans not in ('y', 'yes'):
            print('Cancelled.')
            sys.exit(0)

    conf.unlink()
    print(f'Config "{project}" deleted.')


def main():
    parser = argparse.ArgumentParser(
        description=f'Workspace Manager v{VERSION} (Python)',
    )
    parser.add_argument(
        'command', nargs='?',
        choices=['mount', 'm', 'unmount', 'u', 'run', 'r',
                 'connect', 'c', 'ssh', 'delete', 'del', 'rm'],
        help='Command to execute',
    )
    parser.add_argument('project', nargs='?', help='Project name')
    parser.add_argument('--list', '-l', action='store_true',
                        help='List available projects')
    parser.add_argument('--force', '-f', action='store_true',
                        help='Skip confirmation (for delete)')

    args = parser.parse_args()

    if args.list:
        projects = list_projects()
        if projects:
            print('\n'.join(projects))
        else:
            print('No projects configured.', file=sys.stderr)
        sys.exit(0)

    if not args.command:
        parser.print_help()
        sys.exit(0)

    if not args.project:
        print('Error: project name required', file=sys.stderr)
        sys.exit(1)

    conf = CONF_DIR / f'{args.project}.conf'
    if not conf.exists():
        print(f'Error: Project config {args.project!r} not found in {CONF_DIR}',
              file=sys.stderr)
        sys.exit(1)

    remote_path = parse_toml(conf, 'remote_path')
    local_mount = parse_toml(conf, 'local_mount')
    editor_cmd = parse_toml(conf, 'editor_cmd')

    cmd = args.command

    if cmd in ('mount', 'm'):
        cmd_mount(remote_path, local_mount)
    elif cmd in ('run', 'r'):
        cmd_run(remote_path, local_mount, editor_cmd, args.project)
    elif cmd in ('unmount', 'u'):
        cmd_unmount(local_mount)
    elif cmd in ('connect', 'c', 'ssh'):
        cmd_connect(remote_path, editor_cmd)
    elif cmd in ('delete', 'del', 'rm'):
        cmd_delete(args.project, conf, local_mount, args.force)


if __name__ == '__main__':
    main()
