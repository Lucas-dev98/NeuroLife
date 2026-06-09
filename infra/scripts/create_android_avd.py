import os
import subprocess
from pathlib import Path

SDK_ROOT = Path(r"C:\Users\lucas.bastos\Downloads\Android\Sdk")
JAVA_HOME = Path(r"C:\Users\lucas.bastos\Downloads\Java\jdk-17")
AVDMANAGER = SDK_ROOT / "cmdline-tools" / "latest" / "bin" / "avdmanager.bat"
SDKMANAGER = SDK_ROOT / "cmdline-tools" / "latest" / "bin" / "sdkmanager.bat"
SYSTEM_IMAGE = "system-images;android-35;google_apis;x86_64"
AVD_NAME = "Pixel_API_35"


def main() -> None:
    os.environ["JAVA_HOME"] = str(JAVA_HOME)
    os.environ["PATH"] = str(JAVA_HOME / "bin") + os.pathsep + os.environ.get("PATH", "")

    subprocess.run(
        [
            str(SDKMANAGER),
            f"--sdk_root={SDK_ROOT}",
            "--install",
            SYSTEM_IMAGE,
        ],
        check=False,
    )

    listed = subprocess.run(
        [str(AVDMANAGER), f"--sdk_root={SDK_ROOT}", "list", "avd"],
        text=True,
        capture_output=True,
        check=False,
    )
    if f"Name: {AVD_NAME}" in listed.stdout:
        print(f"AVD {AVD_NAME} ja existe.")
        return

    created = subprocess.run(
        [
            str(AVDMANAGER),
            f"--sdk_root={SDK_ROOT}",
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
        ],
        input="no\n",
        text=True,
        check=False,
    )
    if created.returncode != 0:
        raise SystemExit(created.returncode)

    print(f"AVD {AVD_NAME} criado com sucesso.")


if __name__ == "__main__":
    main()
