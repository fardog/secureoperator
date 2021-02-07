@echo off

:_____________________uac

:%1 mshta vbscript:CreateObject("Shell.Application").ShellExecute("cmd.exe","/c %~s0 ::","","runas",1)(window.close)&&exit

:_____________________uac_end

:_____________________main

:main
cls

cd %~dp0

set servicename=dohproxy
set app="%~dp0\doh-proxy_wrapper.exe"
set args="-http2 -endpoint https://ENDPOINT -google -endpoint-ips IPS -edns-subnet auto -listen :53 -cache=true -loglevel info"
set action=
set command=

echo=
echo 1. Install doh-proxy service
echo=
echo 2. Restart doh-proxy service
echo=
echo 3. Edit doh-proxy service
echo=
echo 4. Uninstall doh-proxy service
echo=
echo 5. Custom command
echo=
echo z. Quit
echo=

set /p action=Please input action number [ e.g. 1 ]: 
echo=
call :do%action%
if %errorlevel%==1 goto main

:_____________________main_end


:_____________________install
:do1
"%~dp0\nssm.exe" install %servicename% %app% "%args%"
echo=
"%~dp0\nssm.exe" set %servicename% AppStdout "%~dp0\doh-proxy.log"
echo=
"%~dp0\nssm.exe" set %servicename% AppStderr "%~dp0\doh-proxy.log"
echo=
"%~dp0\nssm.exe" set %servicename% AppRotateFiles 1
echo=
"%~dp0\nssm.exe" set %servicename% AppRotateBytes 10485760
echo=
"%~dp0\nssm.exe" start %servicename%
echo=
pause
echo=
goto main
:_____________________install_end


:_____________________restart
:do2
echo=
"%~dp0\nssm.exe" restart %servicename%
echo=
pause
goto main
:_____________________restart_end

:_____________________edit
:do3
echo=
"%~dp0\nssm.exe" edit %servicename%
echo=
"%~dp0\nssm.exe" restart %servicename%
echo=
pause
goto main
:_____________________edit_end

:_____________________remove
:do4

echo=
"%~dp0\nssm.exe" stop %servicename%
echo=
"%~dp0\nssm.exe" remove %servicename%
echo=
pause
goto main
:_____________________remove_end

:_____________________command
:do5

echo=
"%~dp0\nssm.exe" /?
echo=
set /p command=Please input command [ e.g. install ] : 
echo=
"%~dp0\nssm.exe" %command%
echo=
pause
goto main
:_____________________command_end

:_____________________EOF
:doz
exit