# How To Run the Open Source Documentation Toolchain

## Definitions
The ${variable} tags in the process below refer to the following. It
is recommended that you set these as shell variables then copy and
paste the commands below.


| Variable | Description                                                                                                                 | Example                                             |
|:--------:|:---------------------------------------------------------------------------------------------------------------------------:|:---------------------------------------------------:|
|  source  |               The place you want to grab copyright notices from. In the form of SSH info for login: user@host               |               source=root@someServer.com            |
|   HOME   |                                   shell variable containing the home directory of the user                                  |                                                     |
|  dstdir  | Absolute pathname on the Target system which will hold the artifacts. This directory must exist before running getallsrc.sh |              dstdir=${HOME}/FolderName              |
|  target  |                              Destination user, host and directory where to hold the artifacts.                              |          target=user@yourServer.com:${dstdir}       |
|  ostpath |                              The path to the extracted distribution of the open source toolchain                            | (while in executable file directory) ostpath=$(pwd) |


## Process

This process would desirably be done from a shell window on the target system.
This is formally defined below, but generally, the taget system is a Linux or
Mac OSX system with the open source toolchain available, and which has plenty
of space to run the toolchain (10's of GB).

Note that if you set each variable above as environment variables in the shell
you can then run the following commands by simply copy-pasting the commands
below.

  * Fetch source packages:
    * If the source host has ssh connectivity to the target host.

          mkdir -p ${dstdir}
          scp ${ostpath}/getallsrc.sh ${source}:
          ssh -t ${source} target=${target} /bin/bash
          ./getallsrc.sh ${target}			# answer questions as necessary
    * If the source host does not have ssh connectivity to the target host
       * Replace the hostname in ${target} with "localhost", and use modified commands
       * Note: 54321 can be anything as long as it is greater than 4096 and not already in use. If the target is running ssh on a nonstandard port, replace 22 with that port number.

             mkdir -p ${dstdir}
             scp ${ostpath}/getallsrc.sh ${source}:
             ssh -R 54321:localhost:22 -t ${source} target=${target} /bin/bash
             ./getallsrc.sh -p 54321 ${target}              # answer questions as necessary

  * Generate Notices:

        PATH=${PATH}:${ostpath}
        cd ${dstdir}
        extract.sh
        mkarchive.sh
        mknotices.sh $(pwd)/Archive $(pwd)/FolderForPackageFiles
