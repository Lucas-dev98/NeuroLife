import os
import subprocess
from pathlib import Path

SDK_ROOT = Path(r"C:\Users\lucas.bastos\Downloads\Android\Sdk")
JAVA_HOME = Path(r"C:\Users\lucas.bastos\Downloads\Java\jdk-17")
SDKMANAGER = SDK_ROOT / "cmdline-tools" / "latest" / "bin" / "sdkmanager.bat"

os.environ["JAVA_HOME"] = str(JAVA_HOME)
os.environ["PATH"] = str(JAVA_HOME / "bin") + os.pathsep + os.environ.get("PATH", "")

result = subprocess.run(
    [str(SDKMANAGER), f"--sdk_root={SDK_ROOT}", "--list_installed"],
    text=True,
    capture_output=True,
    check=False,
)
print(result.stdout)
