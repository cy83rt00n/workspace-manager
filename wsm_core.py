#!/usr/bin/env python3
"""WSM Workspace Manager — shared business logic."""

import re
import subprocess
from pathlib import Path

CONF_DIR = Path.home() / '.config' / 'workspace'
VERSION = '2.0.0'


def parse_toml(filepath, key):
    """Parse TOML key = "value" from config file."""
    try:
        with open(filepath) as f:
            for line in f:
                m = re.match(rf'^{key}\s*=\s*"(.+)"', line)
                if m:
                    return m.group(1)
    except FileNotFoundError:
        pass
    return ''


def is_mounted(path):
    """Check if path is a mountpoint."""
    if not path:
        return False
    return subprocess.run(
        ['mountpoint', '-q', path], capture_output=True
    ).returncode == 0


def check_net(alias):
    """Resolve SSH alias and verify host reachability via nc.
    Returns (ok: bool, error: str)."""
    try:
        result = subprocess.run(
            ['ssh', '-G', alias], capture_output=True, text=True, timeout=5
        )
        ssh_info = result.stdout
    except Exception:
        return False, f'SSH alias {alias!r} not found in ~/.ssh/config'

    host = ''
    port = '22'
    for line in ssh_info.split('\n'):
        if re.match(r'^hostname\s+', line, re.IGNORECASE):
            host = line.split(None, 1)[1]
        if re.match(r'^port\s+', line, re.IGNORECASE):
            port = line.split(None, 1)[1]

    if not host:
        return False, f'Cannot resolve hostname for {alias!r}'

    result = subprocess.run(
        ['nc', '-z', '-w', '2', host, port],
        capture_output=True, timeout=5,
    )
    if result.returncode != 0:
        return False, f'Host {host}:{port} is unreachable. Check network or VPN.'
    return True, ''


def load_projects():
    """Load all projects with their config and mount status."""
    projects = []
    if not CONF_DIR.exists():
        return projects
    for conf in sorted(CONF_DIR.glob('*.config.toml')):
        name = conf.stem
        local_mount = parse_toml(conf, 'local_mount')
        remote_path = parse_toml(conf, 'remote_path')
        editor_cmd = parse_toml(conf, 'editor_cmd')
        mounted = bool(local_mount and is_mounted(local_mount))
        projects.append({
            'name': name,
            'remote_path': remote_path,
            'local_mount': local_mount,
            'editor_cmd': editor_cmd,
            'mounted': mounted,
            'conf': str(conf),
        })
    return projects


def wsm_cli(action, project):
    """Call the CLI module as a subprocess."""
    import sys
    cli_path = Path(__file__).resolve().parent / 'wsm_cli.py'
    result = subprocess.run(
        [sys.executable, str(cli_path), action, project],
        capture_output=True, text=True, timeout=60,
    )
    return result.stdout, result.stderr, result.returncode


def match_key(key, *chars):
    """Match get_wch() code against any of the given characters (any layout)."""
    for c in chars:
        if key == ord(c):
            return True
    if key > 255:
        try:
            if chr(key) in chars:
                return True
        except (ValueError, OverflowError):
            pass
    return False


def validate_config(alias, remote_path, local_mount):
    """Validate config fields.  Returns (ok: bool, error: str)."""
    alias = ''.join(c for c in alias if c.isalnum() or c in '_-')
    if not alias:
        return False, 'Alias required (a-z, 0-9, _, -)'
    if remote_path.count(':') != 1 or not remote_path.split(':')[0]:
        return False, 'Remote must be alias:/path'
    if not local_mount.startswith('/'):
        return False, 'Local mount must be absolute path'
    return True, ''


def save_config(alias, remote_path, local_mount, editor_cmd='zed',
                old_conf=None):
    """Write project config file.  If old_conf given and alias changed, delete old."""
    CONF_DIR.mkdir(parents=True, exist_ok=True)
    new_conf = CONF_DIR / f'{alias}.config.toml'
    if old_conf and str(old_conf) != str(new_conf) and Path(old_conf).exists():
        Path(old_conf).unlink()
    with open(new_conf, 'w') as f:
        f.write(f'remote_path = "{remote_path}"\n')
        f.write(f'local_mount = "{local_mount}"\n')
        f.write(f'editor_cmd = "{editor_cmd}"\n')
    return new_conf


def can_delete_config(conf_path, local_mount):
    """Check if config can be safely deleted.  Returns (ok: bool, reason: str)."""
    if not Path(conf_path).exists():
        return False, f'Config not found: {conf_path}'
    if local_mount and is_mounted(local_mount):
        return False, 'Project is mounted. Unmount first.'
    return True, ''


def delete_config_file(conf_path, local_mount):
    """Delete config file.  Raises ValueError if not safe."""
    ok, reason = can_delete_config(conf_path, local_mount)
    if not ok:
        raise ValueError(reason)
    Path(conf_path).unlink()


def generate_keypair(name):
    """Generate ED25519 SSH keypair.  Returns (private_path, public_path)."""
    import subprocess
    clean = Path(name).stem
    clean = re.sub(r'\.(key|pub)$', '', clean)
    if not clean:
        raise ValueError('Invalid key name')
    private = CONF_DIR / f'{clean}.key'
    public = CONF_DIR / f'{clean}.pub'
    if private.exists():
        raise FileExistsError(f'{private} already exists')
    subprocess.run(
        ['ssh-keygen', '-t', 'ed25519', '-f', str(private),
         '-C', f'wsm-{clean}', '-N', ''],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )
    subprocess.run(
        ['mv', f'{private}.pub', str(public)],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )
    return str(private), str(public)


def desktop_snippet_text(project):
    """Generate XDG Desktop Action snippet for a project."""
    return [
        f'Actions={project};', '',
        f'[Desktop Action {project}]',
        f'Name=Open Remote: {project}',
        f'Exec=bash -c \'source $HOME/.wsm && wsm run {project}\'',
        f'Identifier={project}',
    ]
