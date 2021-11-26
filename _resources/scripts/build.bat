@echo off
docker build -t gobins --build-arg GIT_TOKEN=%GIT_AUTH_FOR_DOCKER% .
docker run --name="gobin_run" gobins
docker cp gobin_run:gobin/_bin/. ./_bin/.
docker rm gobin_run
docker rmi gobins
copy ".\_bin\yt_downloader_windows_amd64.exe" ".\_bin\yt_downloader.exe"
echo "BAT done"