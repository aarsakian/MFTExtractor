MFTExtractor
============

### A Parser  of ~~Master File Table~~  NTFS file system.



Using this tool you can explore ~~$MFT~~ NTFS and its file system attributes. You can selectively extract filesystem information of record  or for a range of records. In addition, you can export the contents of files. 

Exporting files can be achieved either by mounting the evidence and providing its physical drive order and partition number or by using the acquired forensic image (Expert Witness Format), or virtual machine disk format. 

#### Examples #####
you can now explore NTFS by providing physical drive number and partition number 
e.g. *-physicaldrive 0 -partition 1* translates to \\\\.\\PHYSICALDRIVE0 D drive respectively,


or by using as input an expert witness format image 
e.g. *-evidence path_to_evidence -partition 1*.

Usage information  type: MFTExtractor  -h
  -MFT string
        absolute path to the MFT file
        
  -attributes string
        show attributes (write any for all attributes)
        
  -deleted
        show deleted records
        
  -entries string
        select particular MFT entries, use comma as a seperator.
        
  -evidence string
        path to image file (EWF formats are supported)
        
  -extensions string
        search MFT records by extensions use , for each extension
        
  -filenames string
        files to export use comma for each file
        
  -filesize
        show file size of a record holding a file
        
  -fromEntry int
        select entry to start parsing (default -1)
        
  -hash string
        select hash md5 or sha1 for exported files.
        
  -index
        show index structures
        
  -listpartitions
        list partitions
        
  -location string
        the path to export  files
        
  -log
        enable logging
        
  -orphans
        show information only for orphan records
        
  -parent
        show information about parent record
        
  -partition int
        select partition number (default -1)
        
  -path string
        base path of files to exported must be absolute e.g. C:\MYFILES\ABC translates to MYFILES\ABC
        
  -physicaldrive int
        select disk drive number for extraction of non resident files (default -1)
        
  -resident
        check whether entry is resident
        
  -runlist
        show runlist of MFT record attributes
        
  -showfilename string
        show the name of the filename attribute of each MFT record choices: Any, Win32, Dos
        
  -showfull
        show full information about record
        
  -showpath
        show the full path of the selected files.
        
  -showtree
        show tree
        
  -showusn
        show information about usnjrnl records
        
  -strategy string
        what strategy will use for files sharing the same file name supported is use Id default is ovewrite. (default "overwrite")
        
  -timestamps
        show all timestamps
        
  -toEntry int
        select entry to end parsing (default 4294967295)
        
  -tree
        reconstrut entries tree
        
  -unallocated
        collect unallocated area of a file system
        
  -usnjrnl
        show information about changes to files and folders.
        
  -vcns
        show the vncs of non resident attributes
        
  -vmdk string
        path to vmdk file (Sparse formats are supported)

