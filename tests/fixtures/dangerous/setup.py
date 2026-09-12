# FIXTURE ONLY - never executed. Inert strings chosen to trip the scanner's
# critical and high rules deterministically.
import os

AWS_KEY = "AKIAIOSFODNN7EXAMPLE"          # critical: AWS Access Key ID

os.system("echo pwned")                    # high: subprocess execution


def go():
    exec("print('x')")                     # high: exec()
