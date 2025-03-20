echo "go proto"
@.\protoc.exe --go_out=.. lobby.proto
@.\protoc.exe --go_out=.. types.proto
@.\protoc.exe --go_out=.. room.proto
@.\protoc.exe --go_out=.. table.proto
@.\protoc.exe --go_out=.. gameddz.proto
@.\protoc.exe --go_out=.. list.proto
@.\protoc.exe --go_out=.. property.proto

pause
