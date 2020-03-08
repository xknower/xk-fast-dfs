echo off
CHCP 65001
REM 注释
echo run building ...
set build_env=%1
set env_version=%2

set Name=xfs

REM SET VERSION INFO
set BIN_VERSION=0.0.0
set BUILD_TIME=00:00:00
set GO_VERSION=1.12.5
set GIT_VERSION=2.21.0.windows.1
set M1="-w -s -X 'main.VERSION=%BIN_VERSION%' -X 'main.BUILD_TIME=%BUILD_TIME%' -X 'main.GO_VERSION=%GO_VERSION%' -X 'main.GIT_VERSION=%GIT_VERSION%'"
echo %M1%

if "%build_env%" == "win" (
  goto win
) else if "%build_env%" == "linux"  (
  goto linux
) else if "%build_env%" == "clean"  (
  goto clean
)  else if "%build_env%" == "all"  (
  goto win
  goto linux
) else (
  goto win
)

REM windows
: win
  echo build windows
  set GOOS=windows

  del %Name%.exe
  del %Name%_x32.exe

  if "%env_version%" == "386" (
    set GOARCH=386
    go build -o %Name%_x32.exe -ldflags %M1% xk.go
  ) else if "%build_env%" == "amd64"  (
    set GOARCH=amd64
    go build -o %Name%.exe -ldflags %M1% xk.go
  ) else  if "%build_env%" == "all" (
    set GOARCH=386
    go build -o %Name%_x32.exe -ldflags %M1% xk.go

    set GOARCH=amd64
    go build -o %Name%.exe -ldflags %M1% xk.go
  ) else (
    set GOOS=windows
    set GOARCH=amd64
    go build -o %Name%.exe -ldflags %M1% xk.go
  )
goto:eof

REM linux
: linux

  echo build linux
  set GOOS=linux

  del %Name%
  del %Name%_x32

 if "%env_version%" == "386" (
   set GOARCH=386
   go build -o %Name%_x32 -ldflags %M1% xk.go
  ) else if "%build_env%" == "amd64"  (
    set GOARCH=386
    go build -o %Name%_x32 -ldflags %M1% xk.go

    set GOARCH=amd64
    go build -o %Name% -ldflags %M1% xk.go
  ) else  if "%build_env%" == "all" (
    set GOARCH=386
    go build -o %Name%_x32 -ldflags %M1% xk.go

    set GOARCH=amd64
    go build -o %Name% -ldflags %M1% xk.go
  ) else (
   set GOARCH=amd64
   go build -o %Name% -ldflags %M1% xk.go
  )
goto:eof

REM mac
: mac
  echo build mac
  set GOOS=darwin

  del %Name%_darwin
  del %Name%_darwin_x32

  if "%env_version%" == "386" (
    set GOARCH=386
    go build -o %Name%_x32 -ldflags %M1% xk.go
  ) else if "%build_env%" == "amd64"  (

  ) else  if "%build_env%" == "all" (
    set GOARCH=386
    go build -o %Name%_x32 -ldflags %M1% xk.go

    set GOARCH=amd64
    go build -o %Name%_darwin -ldflags %M1% xk.go
  ) else (
    set GOARCH=amd64
    go build -o %Name%_darwin -ldflags %M1% xk.go
 )
goto:eof

REM clean build file
: clean
  del %Name%.exe
  del %Name%_x32.exe
  del %Name%
  del %Name%_x32
goto:eof

echo success ...