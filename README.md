MFTExtractor
============

### A Parser  of Master File Table written  in go.



a tool to explore $MFT and its attributes

Usage information  type: MFTExtractor  -h

  -MFT string
    	absolute path to the MFT file (default "MFT file")

  -attributes string
    	show attributes

  -entry int
    	select a particular MFT entry (default -1)

  -export string
    	export resident files (default "None")

  -fileName string
    	show the name of the filename attribute of each MFT record choices: Any, Win32, Dos

  -filesize
    	show file size of a record holding a file

  -fromEntry int
    	select entry to start parsing

  -index
    	show index structures

  -physicalDrive string (offset of volume is hardcoded for the moment)
    	use physical drive information for extraction of non resident files

  -resident
    	check whether entry is resident

  -runlist
    	show runlist of MFT record attributes

  -structure
    	reconstrut entries tree

  -timestamps
    	show all timestamps

  -toEntry int
    	select entry to end parsing 

  -vcns
    	show the vncs of non resident attributes


