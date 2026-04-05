# ZFS Rebuild And SSH Takeover Runbook

Status: Draft operator runbook
Date: 2026-04-04

This runbook is for the tiny slice of work that still has to be done manually by a human at the keyboard.

It is written for an operator with no Linux background.

The goal is not to finish Moltbox manually. The goal is only to:

1. reinstall the host cleanly
2. get the host reachable over SSH again
3. stop touching it so the AI builder can take over

If you reach working SSH, this runbook is done.

Related records:

- [`../plans/2026-04-04-clean-moltbox-execution-plan.md`](../plans/2026-04-04-clean-moltbox-execution-plan.md)
- [`../plans/2026-04-04-clean-moltbox-validation-plan.md`](../plans/2026-04-04-clean-moltbox-validation-plan.md)
- [`../plans/2026-04-04-clean-moltbox-builder-prompt.md`](../plans/2026-04-04-clean-moltbox-builder-prompt.md)

## What This Runbook Does Not Do

This runbook does not:

- build the final ZFS dataset layout
- deploy Docker containers
- install or restore Moltbox services
- create AI service users
- restore OpenClaw
- configure backups or patching

Those are AI tasks after SSH access works.

## Stop Condition

Stop this runbook immediately after both of these are true:

- you can SSH into the rebuilt host from your Windows workstation
- the AI builder can use that SSH access

After that, do not keep typing Linux commands unless the AI explicitly asks for one more manual step.

## Known Machine Facts

Use these facts to avoid wiping the wrong disk:

- Metal host name: `moltbox-prime`
- Current LAN IP before rebuild: `192.168.1.189`
- Current primary wired interface: `enp6s0`
- Current OS family: Ubuntu Server 24.04.4 LTS

Current disks seen on the live host:

| Device | Size | Model | Meaning |
| --- | --- | --- | --- |
| `/dev/sda` | 12.7T | `ST14000NM0018-2H` | Backup disk. Do not wipe this. |
| `/dev/nvme0n1` | 1.8T | `Samsung SSD 990 PRO 2TB` | Leave this alone during reinstall unless the AI explicitly told you otherwise. |
| `/dev/nvme1n1` | 1.8T | `CT2000P310SSD8` | Current OS disk. This is the default reinstall target. |

## Hard Rules

- Do not wipe `/dev/sda`.
- Do not store keys in Git.
- Do not put runtime or service files under a home directory for normal operation.
- Do not keep going after SSH works just because you think you should "finish setup."
- If the screen in front of you does not match this runbook closely, stop and ask before wiping anything.

## What You Need Before You Start

You need all of these:

- physical or remote console access to `moltbox-prime`
- an Ubuntu Server 24.04 LTS installer USB
- a working keyboard and monitor, or equivalent remote KVM
- the machine connected to the network by Ethernet
- your Windows workstation with the existing SSH key files

Your current workstation SSH files already exist here:

- `C:\Users\Jason\.ssh\id_ed25519`
- `C:\Users\Jason\.ssh\id_ed25519.pub`

The current SSH alias file is here:

- `C:\Users\Jason\.ssh\config`

## Before You Wipe Anything

Do not start the reinstall until the AI builder has already confirmed all extraction and backup work is complete.

The AI must finish these tasks first:

- copy host-state backups to `/mnt/moltbox-backup`
- create verified OpenClaw backups for `test` and `prod`
- record current live appliance facts

If you are not sure whether that happened, stop and ask the AI builder. Do not guess.

## Phase 1: Reinstall Ubuntu

This is the part that is unavoidably manual.

### Step 1: Boot The Installer

1. Insert the Ubuntu Server 24.04 LTS USB.
2. Reboot the machine.
3. Boot from the USB installer.
4. Choose the normal Ubuntu Server install option.

### Step 2: Click Through The Installer Carefully

Use these choices unless a screen makes that impossible.

Language and keyboard:

- use English
- use the normal US keyboard layout

Network:

- use the wired interface
- DHCP is fine
- do not try to hand-configure static IP right now

Proxy and mirror:

- leave proxy blank
- use the default Ubuntu mirror

### Step 3: Choose The Install Disk

This is the dangerous part.

Pick only the current OS disk:

- `/dev/nvme1n1`
- model `CT2000P310SSD8`
- size about `1.8T`

Do not touch:

- `/dev/sda`
- model `ST14000NM0018-2H`
- size about `12.7T`

Also leave `/dev/nvme0n1` alone unless the AI explicitly told you to do something else.

If the installer screen does not make it completely obvious which disk is which, stop. Do not guess.

### Step 4: Create The Bootstrap Admin Account

Create one normal Ubuntu admin user during install.

Use these values unless you have a strong reason not to:

- Your name: `Jason Pekovitch`
- Server name: `moltbox-prime`
- Username: `jpekovitch`

Why this matters:

- it keeps your existing human admin path alive
- it keeps the current local SSH workflow close to working
- it does not make this account part of the final runtime ownership model

This account is for human admin access only. The AI will later move the real runtime and service ownership to system-owned paths and system identities.

### Step 5: Install OpenSSH During Install

When the installer asks whether to install OpenSSH server:

- choose `Yes`

Do not worry about importing an SSH identity during install. We will do that in a simpler way after first boot.

### Step 6: Finish The Install

1. Let the installer complete.
2. Remove the USB when prompted.
3. Reboot into the fresh OS.

## Phase 2: First Boot Commands On The Linux Console

You are now on the rebuilt machine itself.

Log in using the username and password you created during install.

Every command below should be typed exactly as shown and then you should press Enter.

### Step 1: Confirm The Hostname

Run:

```bash
hostnamectl
```

You want to see:

- `Static hostname: moltbox-prime`

If you do not, fix it with:

```bash
sudo hostnamectl set-hostname moltbox-prime
hostnamectl
```

### Step 2: Find The New IP Address

Run:

```bash
ip -brief addr show enp6s0
```

You want to see:

- interface `enp6s0`
- state `UP`
- an IPv4 address that looks like `192.168.1.xxx/24`

Write that IP address down. You will need it on the Windows workstation in a minute.

### Step 3: If There Is No IPv4 Address, Fix DHCP

If the command above does not show an IPv4 address on `enp6s0`, run these commands exactly:

```bash
sudo tee /etc/netplan/01-moltbox-bootstrap.yaml >/dev/null <<'EOF'
network:
  version: 2
  ethernets:
    enp6s0:
      dhcp4: true
EOF
sudo netplan apply
ip -brief addr show enp6s0
```

If you still do not have an IPv4 address, stop here and fix the network before doing anything else.

### Step 4: Make Sure SSH Is Running

First try this:

```bash
sudo systemctl enable --now ssh
systemctl status ssh --no-pager
```

If that works, continue.

If it says the `ssh` service does not exist, run these commands:

```bash
sudo apt update
sudo apt install -y openssh-server
sudo systemctl enable --now ssh
systemctl status ssh --no-pager
```

You want the final status output to include:

- `active (running)`

### Step 5: Optional Firewall Fix If SSH Is Still Blocked

Only do this if SSH still does not work from Windows later.

Run:

```bash
sudo ufw status
sudo ufw allow OpenSSH
```

If `ufw` is not installed or not active, that is fine. Just move on.

## Phase 3: Windows Workstation Commands

Now go back to your Windows workstation.

Open PowerShell.

The goal here is:

- use the password one time
- install your existing public key on the rebuilt host
- confirm key-based SSH works

### Step 1: Confirm Your Existing Key File Exists

Run in PowerShell:

```powershell
Test-Path $env:USERPROFILE\.ssh\id_ed25519.pub
```

You want the answer:

```powershell
True
```

If it says `False`, stop and ask for help before doing anything else.

### Step 2: Set The Variables For This Session

Replace the IP address below if the rebuilt host got a different one.

Run:

```powershell
$HostIp = "192.168.1.189"
$AdminUser = "jpekovitch"
```

If `ip -brief addr show enp6s0` showed a different IP, put that IP into `$HostIp`.

### Step 3: Install Your Public Key Using Password Login

Run:

```powershell
Get-Content $env:USERPROFILE\.ssh\id_ed25519.pub | ssh "$AdminUser@$HostIp" "umask 077; mkdir -p ~/.ssh; cat >> ~/.ssh/authorized_keys"
```

What to expect:

- the first connection may ask whether you trust the host key
- type `yes` and press Enter
- it will then ask for the password you created during install
- after you enter the password, the command should finish quietly

That one command installs your existing workstation public key onto the rebuilt host.

### Step 4: Test Key-Based SSH

Run:

```powershell
ssh -i $env:USERPROFILE\.ssh\id_ed25519 "$AdminUser@$HostIp" "hostname && whoami && ip -brief addr show enp6s0"
```

You want output that shows:

- hostname `moltbox-prime`
- user `jpekovitch`
- the IP address on `enp6s0`

If that works, SSH takeover is working.

### Step 5: Optional Alias Test

If the host kept the same IP and your local SSH config still matches, this may also work:

```powershell
ssh moltbox "hostname && whoami"
```

If that fails, do not panic. The direct SSH command in Step 4 is the real test.

## Phase 4: Hand Control Back To The AI

Once Step 4 above works, stop doing manual Linux setup.

At that point, the only message you should need to send back is something like:

`SSH is restored. Host is reachable as jpekovitch@<new-ip>.`

After that, the AI builder should take over and do the rest:

- install ZFS packages if needed
- create the ZFS pool
- create datasets for `/srv/moltbox-state`, `/srv/moltbox-logs`, and `/var/lib/moltbox`
- verify snapshots and rollback
- recreate system-owned paths
- deploy the clean appliance
- validate the full run

## Troubleshooting

### Problem: I Do Not Know Which Disk To Pick In The Installer

Stop.

Do not continue until you can positively identify these:

- wipe target: `/dev/nvme1n1`, 1.8T, model `CT2000P310SSD8`
- never wipe: `/dev/sda`, 12.7T, model `ST14000NM0018-2H`

### Problem: `ip -brief addr show enp6s0` Shows No IPv4 Address

Run the DHCP fix block from Phase 2, Step 3 exactly as written.

If that still fails, stop and fix network before trying SSH.

### Problem: PowerShell Says The Public Key File Does Not Exist

Run:

```powershell
Get-ChildItem $env:USERPROFILE\.ssh
```

You are looking for:

- `id_ed25519`
- `id_ed25519.pub`

If they are missing, stop and ask for the key material explicitly.

### Problem: SSH Asks For A Password Every Time

That means the key was not installed correctly.

Run the public-key install command from Phase 3, Step 3 again and type the password carefully when prompted.

Then rerun the key-based SSH test from Phase 3, Step 4.

### Problem: SSH Says `Connection Refused`

Go back to the Linux console and run:

```bash
sudo systemctl status ssh --no-pager
```

If it is not running, use the commands from Phase 2, Step 4.

### Problem: SSH Connects But The Hostname Is Wrong

Go back to the Linux console and run:

```bash
sudo hostnamectl set-hostname moltbox-prime
hostnamectl
```

Then reconnect.

## Completion Checklist

This runbook is complete only when all of these are true:

- Ubuntu is reinstalled on `/dev/nvme1n1`
- `/dev/sda` was not wiped
- the host name is `moltbox-prime`
- `enp6s0` has a real IPv4 address
- SSH is running
- your workstation public key is installed
- key-based SSH works from Windows
- you have stopped manual setup and handed the box back to the AI
