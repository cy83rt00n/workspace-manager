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
    for conf in sorted(CONF_DIR.glob('*.conf')):
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
