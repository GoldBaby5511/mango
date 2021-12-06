@echo build and run
@echo build logger
@cd logger
@go build
@start .\logger.exe
@cd ..

@echo build center
@cd center
@go build
@start .\center.exe
@cd ..

@echo build config
@cd config
@go build
@start .\config.exe
@cd ..

@echo build gateway
@cd gateway
@go build
@start .\gateway.exe
@cd ..

@echo build login
@cd login
@go build
@start .\login.exe
@cd ..
