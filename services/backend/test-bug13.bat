@echo off
cd /d D:\ALHudhudAI\services\backend
go test -v -count=1 -run "TestBug13_CacheNotInvalidatedOnChannelOffline_Exploration" ./internal/redis/
