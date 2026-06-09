import os
import subprocess
from pathlib import Path

SDK_ROOT = Path(r"C:\Users\lucas.bastos\Downloads\Android\Sdk")
JAVA_HOME = Path(r"C:\Users\lucas.bastos\Downloads\Java\jdk-17")
AVDMANAGER = SDK_ROOT / "cmdline-tools" / "latest" / "bin" / "avdmanager.bat"

AVD_NAME = "Pixel_API_35_Default_B"
SYSTEM_IMAGE = "system-images;android-35;default;x86_64"

os.environ["JAVA_HOME"] = str(JAVA_HOME)
os.environ["ANDROID_SDK_ROOT"] = str(SDK_ROOT)
os.environ["ANDROID_HOME"] = str(SDK_ROOT)
os.environ["PATH"] = str(JAVA_HOME / "bin") + os.pathsep + os.environ.get("PATH", "")

listed = subprocess.run([str(AVDMANAGER), "list", "avd"], text=True, capture_output=True, check=False)
if f"Name: {AVD_NAME}" not in listed.stdout:
    subprocess.run([
        str(AVDMANAGER),
        "create",
        "avd",
        "--name",
        AVD_NAME,
        "--package",
        SYSTEM_IMAGE,
        "--device",
        "pixel",
        "--sdcard",
        "1024M",
    ], input="no\n", text=True, check=False)
print(AVD_NAME)
