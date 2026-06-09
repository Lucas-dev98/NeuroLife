import os
import subprocess
from pathlib import Path

SDK_ROOT = Path(r"C:\Users\lucas.bastos\Downloads\Android\Sdk")
JAVA_HOME = Path(r"C:\Users\lucas.bastos\Downloads\Java\jdk-17")
SDKMANAGER = SDK_ROOT / "cmdline-tools" / "latest" / "bin" / "sdkmanager.bat"

CORE_PACKAGES = [
    "platform-tools",
    "platforms;android-35",
    "build-tools;35.0.0",
    "emulator",
]


def run(cmd: str, stdin_text: str | None = None) -> None:
    completed = subprocess.run(cmd, input=stdin_text, text=True, shell=True)
    if completed.returncode != 0:
        raise SystemExit(completed.returncode)


def main() -> None:
    if not SDKMANAGER.exists():
        raise SystemExit(f"sdkmanager not found: {SDKMANAGER}")
    if not (JAVA_HOME / "bin" / "java.exe").exists():
        raise SystemExit(f"java.exe not found: {JAVA_HOME / 'bin' / 'java.exe'}")

    os.environ["JAVA_HOME"] = str(JAVA_HOME)
    os.environ["PATH"] = str(JAVA_HOME / "bin") + os.pathsep + os.environ.get("PATH", "")

    sdk = str(SDK_ROOT)
    sm = str(SDKMANAGER)

    run(f'"{sm}" --sdk_root="{sdk}" --licenses', stdin_text="y\n" * 300)
    pkg_args = " ".join(f'"{pkg}"' for pkg in CORE_PACKAGES)
    run(f'"{sm}" --sdk_root="{sdk}" --install {pkg_args}')
    print("Android SDK core instalado com sucesso.")


if __name__ == "__main__":
    main()
