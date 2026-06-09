import os
import subprocess
from pathlib import Path

SDK_ROOT = Path(r"C:\Users\lucas.bastos\Downloads\Android\Sdk")
JAVA_HOME = Path(r"C:\Users\lucas.bastos\Downloads\Java\jdk-17")
SDKMANAGER = SDK_ROOT / "cmdline-tools" / "latest" / "bin" / "sdkmanager.bat"
PACKAGES = ["platforms;android-36", "build-tools;28.0.3"]


def run(cmd: str) -> None:
    completed = subprocess.run(cmd, text=True, shell=True)
    if completed.returncode != 0:
        raise SystemExit(completed.returncode)


def main() -> None:
    os.environ["JAVA_HOME"] = str(JAVA_HOME)
    os.environ["PATH"] = str(JAVA_HOME / "bin") + os.pathsep + os.environ.get("PATH", "")
    sdk = str(SDK_ROOT)
    sm = str(SDKMANAGER)
    pkg_args = " ".join(f'"{pkg}"' for pkg in PACKAGES)
    run(f'"{sm}" --sdk_root="{sdk}" --install {pkg_args}')
    print("Pacotes exigidos pelo Flutter instalados.")


if __name__ == "__main__":
    main()
