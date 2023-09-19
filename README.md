MFTExtractor
============

### A Parser  of ~~Master File Table~~  NTFS file system.



Using this tool you can explore $MFT and its attributes. You can selectively extract information about an entry or a range of entries. In addition, you can export the contents of the file if you have mounted the evidence and provide its physical drive order and partition number by using the respective parameters. 

**New**  you can now explore $MFT by providing physicalDrive and partitionNumber
e.g. -physicalDrive 0 -partitionNumber 1 translates to \\\\.\\PHYSICALDRIVE0 D drive.

Usage information  type: MFTExtractor  -h

-MFT string
        absolute path to the MFT file (default "Disk MFT")

  -attributes string
        show attributes

  -entry int
        select a particular MFT entry (default -1)
        
  -evidence string
        path to image file

  -extension string
        search MFT records by extension

  -filename string
        file to export

  -filesize
        show file size of a record holding a file

  -fromEntry int
        select entry to start parsing (default -1)

  -index
        show index structures

  -listpartitions
        list partitions

  -location string
        the path to export  files

  -partition int
        select partition number (default -1)

  -physicaldrive int
        select disk drive number for extraction of non resident files (default -1)

  -resident
        check whether entry is resident

  -runlist
        show runlist of MFT record attributes

  -showfilename string
        show the name of the filename attribute of each MFT record choices: Any, Win32, Dos

  -structure
        reconstrut entries tree

  -timestamps
        show all timestamps

  -toEntry int
        select entry to end parsing (default 4294967295)

  -vcns
        show the vncs of non resident attributes
        