# Package Distribution Guide

This guide covers distributing Markdown in the Middle to package managers (Homebrew, apt, Chocolatey). This file is not checked into the repository—it's for reference when you're ready to release to package managers.

## Homebrew (macOS)

### Step 1: Create a Tap (Custom Repository)

```bash
git clone https://github.com/rickcrawford/homebrew-markdowninthemiddle.git
cd homebrew-markdowninthemiddle
```

### Step 2: Create the Formula File

Create `Formula/markdowninthemiddle.rb`:

```ruby
class Markdowninthemiddle < Formula
  desc "HTTPS forward proxy that converts HTML to Markdown"
  homepage "https://github.com/rickcrawford/markdowninthemiddle"
  url "https://github.com/rickcrawford/markdowninthemiddle/archive/v1.0.0.tar.gz"
  sha256 "..."  # Run: shasum -a 256 <downloaded_file>
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-ldflags", "-s -w", "-o", bin/"markdowninthemiddle", "."
  end

  test do
    system bin/"markdowninthemiddle", "--version"
  end
end
```

### Step 3: Test Locally

```bash
brew install --build-from-source ./Formula/markdowninthemiddle.rb
```

### Step 4: Push to Tap

Users can then install with:

```bash
brew tap rickcrawford/markdowninthemiddle
brew install markdowninthemiddle
```

---

## APT (Debian/Ubuntu)

### Step 1: Create Pre-built Binaries

Create a release in GitHub Releases with pre-built binaries (e.g., `markdowninthemiddle-1.0.0-amd64.deb`).

### Step 2: Generate DEB Package

Use `fpm` to generate DEB:

```bash
go build -o markdowninthemiddle ./cmd
fpm -s dir -t deb -n markdowninthemiddle -v 1.0.0 \
  -a amd64 --prefix=/usr/local/bin markdowninthemiddle
```

### Step 3: Host in a PPA

Upload to Launchpad PPA:

```bash
# Sign and upload
dput ppa:rickcrawford/markdowninthemiddle markdowninthemiddle_1.0.0_source.changes
```

### Step 4: Users Install

```bash
sudo add-apt-repository ppa:rickcrawford/markdowninthemiddle
sudo apt update
sudo apt install markdowninthemiddle
```

---

## Chocolatey (Windows)

### Step 1: Create nuspec File

Create `markdowninthemiddle.nuspec`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata>
    <id>markdowninthemiddle</id>
    <version>1.0.0</version>
    <title>Markdown in the Middle</title>
    <authors>Rick Crawford</authors>
    <description>HTTPS forward proxy that converts HTML to Markdown</description>
    <projectUrl>https://github.com/rickcrawford/markdowninthemiddle</projectUrl>
    <licenseUrl>https://github.com/rickcrawford/markdowninthemiddle/blob/main/LICENSE</licenseUrl>
  </metadata>
  <files>
    <file src="markdowninthemiddle.exe" target="tools"/>
  </files>
</package>
```

### Step 2: Build Package

```bash
choco pack markdowninthemiddle.nuspec
```

### Step 3: Submit to Chocolatey

Submit to Chocolatey community feed or host privately:

```bash
choco push markdowninthemiddle.1.0.0.nupkg --source https://push.chocolatey.org/
```

### Step 4: Users Install

```bash
choco install markdowninthemiddle
```

---

## Release Checklist

- [ ] Update version in source code (if applicable)
- [ ] Update CHANGELOG
- [ ] Tag release: `git tag v1.0.0`
- [ ] Create GitHub Release with pre-built binaries
- [ ] Update Homebrew tap formula
- [ ] Build and test APT package
- [ ] Test Homebrew installation locally
- [ ] Submit Chocolatey package
- [ ] Announce release
