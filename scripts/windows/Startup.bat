@echo build and run
@cd ..\..
@if not exist .\cmd\config\configs mkdir .\cmd\config\configs\config
@copy .\configs\config\center.json .\cmd\config\configs\config
@copy .\configs\config\config.json .\cmd\config\configs\config
@copy .\configs\config\gateway-100.json .\cmd\config\configs\config
@copy .\configs\config\list.json .\cmd\config\configs\config
@copy .\configs\config\lobby.json .\cmd\config\configs\config
@copy .\configs\config\property.json .\cmd\config\configs\config
@copy .\configs\config\robot-3000.json .\cmd\config\configs\config
@copy .\configs\config\room-2000.json .\cmd\config\configs\config
@copy .\configs\config\table-1000.json .\cmd\config\configs\config

@echo call build
@call:build logger -Type=1 -Id=1 -CenterAddr="127.0.0.1:10050"
@call:build center -Type=2 -Id=50 -CenterAddr="127.0.0.1:10050"
@call:build config -Type=3 -Id=60 -CenterAddr="127.0.0.1:10050"
@call:build gateway -Type=4 -Id=100 -CenterAddr="127.0.0.1:10050"
@call:build lobby -Type=5 -Id=70 -CenterAddr="127.0.0.1:10050"
@call:build property -Type=6 -Id=80 -CenterAddr="127.0.0.1:10050"
@call:build robot -Type=9 -Id=3000 -CenterAddr="127.0.0.1:10050"
@call:build list -Type=10 -Id=90 -CenterAddr="127.0.0.1:10050"
@call:build table -Type=11 -Id=1000 -CenterAddr="127.0.0.1:10050"
@call:build room -Type=12 -Id=2000 -CenterAddr="127.0.0.1:10050"
@goto:eof

:build
@set appName=%~1
@cd .\cmd\%appName%\
@go build
@start .\%appName%.exe %~2=%~3 %~4=%~5 %~6=%~7
@cd ../..
@goto:eof
