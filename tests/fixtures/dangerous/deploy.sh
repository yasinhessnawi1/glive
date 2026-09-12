#!/bin/sh
# FIXTURE ONLY - never executed.
rm -rf /var/tmp/example                    # high: dangerous file deletion
mkfs.ext4 /dev/null                        # critical: disk formatting command
