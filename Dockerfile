FROM bms-gobin-builder:latest AS builder
WORKDIR /gobin

# Copy our repo
COPY . ./
ARG GIT_TOKEN

# Authentication for private dependencies
RUN git config --global url."https://${GIT_TOKEN}:@github.com/".insteadOf "https://github.com/"
# build everything except macOS 32bit
# rename all
RUN cd _bin && gox -ldflags '-w -s' -osarch '!darwin/386' && for file in _bin*; do mv -v "$file" "${file/_bin/yt_downloader}"; done
# UPX all but allow failures for unsupported executables
RUN cd _bin && for file in yt*; do upx -9 -k $file || true; done
# Remove all tilde postfixed backup files created by UPX
RUN cd _bin && find -name "*~" -delete

CMD ["/bin/sh", "-c", "echo Docker done"]