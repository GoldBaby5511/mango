@echo off & setlocal enabledelayedexpansion
@set /p appName=请输入新服务名,appName=
@cd ..\..
@if not exist .\cmd\%appName% mkdir .\cmd\%appName%
@if not exist .\cmd\%appName%\business mkdir .\cmd\%appName%\business
@cd scripts\template
for /f "tokens=*" %%i in (main.txt) do (
    if "%%i"=="" (echo.) else (set "line=%%i" & call :chg)
)>>..\..\cmd\%appName%\main.go

@cd ..\..
@copy .\scripts\template\business.txt .\cmd\%appName%\business\
@cd .\cmd\%appName%
@go fmt
@cd business
@ren business.txt business.go
pause
exit
:chg 
set "line=!line:template=%appName%!"
echo !line!
goto :eof

