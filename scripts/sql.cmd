@echo off
rem Command Prompt launcher for sql.ps1 (cmd opens .ps1 files in Notepad instead of running them).
rem   scripts\sql "SELECT * FROM stock_master"
rem   scripts\sql -File database\test_update.sql
rem   scripts\sql                                  (interactive)
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0sql.ps1" %*
