#!/bin/sh
# FIXTURE ONLY - four MEDIUM findings and nothing higher, to pin the
# "more than three medium findings" escalation in Scanner.analyzeFindings.
sudo apt-get update
chmod 777 /tmp/example
su - builder
sudo ldconfig
