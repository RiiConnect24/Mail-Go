@echo off
:: Script allowing us to cache Go dependencies in Docker cache.
echo Installing Modules...

echo Installing google/uuid...
go get github.com/google/uuid

echo Installing discordapp/lilliput...
go get github.com/discordapp/lilliput

echo Installing logrusorgru/aurora...
go get github.com/logrusorgru/aurora

echo Installing go-sql-driver/mysql...
go get github.com/go-sql-driver/mysql

echo Installing getsentry/raven-go...
go get github.com/getsentry/raven-go

choice /c yn /m "Modules installed. Do you want to build the Application now?"
if (%errorlevel% == 1) goto install
if (%errorlevel% == 2) goto finish

:install
echo Building...
go build
echo Cleaning...
go clean

:finish
echo Installation done. Press any button to exit...
pause >nul
exit