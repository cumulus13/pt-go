@echo off
call gobuild -o pt.exe .\pt
copy /y pt.exe c:\TOOLS\exe
exit
