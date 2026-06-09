import os
import subprocess
import sys
from pathlib import Path

SDK_ROOT = Path(r"C:\Users\lucas.bastos\Downloads\Android\Sdk")
SDKMANAGER = SDK_ROOT / "cmdline-tools" / "latest" / "bin" / "sdkmanager.bat"
AVDMANAGER = SDK_ROOT / "cmdline-tools" / "latest" / "bin" / "avdmanager.bat"
JAVA_HOME = Path(r"C:\Users\lucas.bastos\Downloads\Java\jdk-17")

PACKAGES = [
    "platform-tools",
    "platforms;android-35",
    "build-tools;35.0.0",
    "emulator",
    "system-images;android-35;google_apis;x86_64",
]


def run(command: str, stdin_text: str | None = None) -> None:
    completed = subprocess.run(
        command,
        input=stdin_text,
        text=True,
        shell=True,
    )
    if completed.returncode != 0:
        raise SystemExit(completed.returncode)


def main() -> None:
    if not SDKMANAGER.exists():
        raise SystemExit(f"sdkmanager not found at: {SDKMANAGER}")
    if not (JAVA_HOME / "bin" / "java.exe").exists():
        raise SystemExit(f"java.exe not found at: {JAVA_HOME / 'bin' / 'java.exe'}")

    os.environ["JAVA_HOME"] = str(JAVA_HOME)
    os.environ["PATH"] = str(JAVA_HOME / "bin") + os.pathsep + os.environ.get("PATH", "")

    sdk = str(SDK_ROOT)
    sdkmanager = str(SDKMANAGER)
    avdmanager = str(AVDMANAGER)

    # Accept all Android SDK licenses non-interactively.
    run(f'"{sdkmanager}" --sdk_root="{sdk}" --licenses', stdin_text="y\n" * 300)

    pkg_args = " ".join(f'"{pkg}"' for pkg in PACKAGES)
    run(f'"{sdkmanager}" --sdk_root="{sdk}" --install {pkg_args}')

    # Create a default emulator profile if it does not already exist.
    avd_list = subprocess.run(
        f'"{avdmanager}" list avd',
        text=True,
        shell=True,
        capture_output=True,
    )
    if "Name: Pixel_API_35" not in avd_list.stdout:
        run(
            (
                f'"{avdmanager}" create avd '
                f'--name "Pixel_API_35" '
                f'--package "system-images;android-35;google_apis;x86_64" '
                f'--device "pixel" '
                f'--sdcard 1024M'
            ),
            stdin_text="no\n",
        )

    print("Android SDK instalado e AVD Pixel_API_35 configurado.")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"Falha na instalacao do Android SDK: {exc}")
        sys.exit(1)
