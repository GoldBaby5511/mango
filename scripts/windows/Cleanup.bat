cd ../../cmd
for /R %%i in (.) do (cd %%i
	del /s *.exe
	del /s *.json
	rd /s /q log
	rd /s /q configs
cd ..)