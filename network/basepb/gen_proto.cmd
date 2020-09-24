 for /r %%s in (*.proto) do (
 	protoc -I. -I%GOPATH%/src -I%GOPATH%/src/github.com/FJSDS/common/network/pbbase --go_out . --proto_path . ./%%~ns.proto
 	CALL :CHECK_FAIL
 )

 :: ///
 :CHECK_FAIL
 @echo off
 if NOT ["%errorlevel%"]==["0"] (
     pause
     exit /b %errorlevel%
 )