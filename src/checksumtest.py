import sys

args = sys.argv[1:]

checksumDict = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ'

def calc_check_digit(filename):

    sumc = int(filename[0:2], 16)
    for i in range(1, 16):
        char = ord(filename[i])
        sumc = (sumc + char) % 256
    return checksumDict[sumc % len(checksumDict)]

for arg in args:
    print(calc_check_digit(arg))
