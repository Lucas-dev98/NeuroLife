import os
import subprocess
from pathlib import Path

SDK_ROOT = Path(r"C:\Users\lucas.bastos\Downloads\Android\Sdk")
JAVA_HOME = Path(r"C:\Users\lucas.bastos\Downloads\Java\jdk-17")
SDKMANAGER = SDK_ROOT / "cmdline-tools" / "latest" / "bin" / "sdkmanager.bat"

PACKAGE = "system-images;android-35;default;x86_64"

os.environ["JAVA_HOME"] = str(JAVA_HOME)
os.environ["PATH"] = str(JAVA_HOME / "bin") + os.pathsep + os.environ.get("PATH", "")

subprocess.run(
    [str(SDKMANAGER), f"--sdk_root={SDK_ROOT}", "--install", PACKAGE],
    check=False,
)
