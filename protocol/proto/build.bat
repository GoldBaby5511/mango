echo "go proto"
@.\protoc.exe --go_out=.. router.proto
@.\protoc.exe --go_out=.. client.proto
@.\protoc.exe --go_out=.. types.proto
@.\protoc.exe --go_out=.. logger.proto

pause
