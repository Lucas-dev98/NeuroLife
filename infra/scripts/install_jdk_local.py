import os
import shutil
import urllib.request
import zipfile
from pathlib import Path

JDK_ROOT = Path(r"C:\Users\lucas.bastos\Downloads\Java")
ZIP_PATH = JDK_ROOT / "jdk17.zip"
EXTRACT_DIR = JDK_ROOT / "_extract"
URL = "https://aka.ms/download-jdk/microsoft-jdk-17.0.12-windows-x64.zip"


def main() -> None:
    JDK_ROOT.mkdir(parents=True, exist_ok=True)
    print("Baixando JDK...")
    urllib.request.urlretrieve(URL, ZIP_PATH)

    print("Extraindo JDK...")
    shutil.rmtree(EXTRACT_DIR, ignore_errors=True)
    EXTRACT_DIR.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(ZIP_PATH) as zf:
        zf.extractall(EXTRACT_DIR)

    extracted = [p for p in EXTRACT_DIR.iterdir() if p.is_dir()]
    if not extracted:
        raise RuntimeError("Pasta do JDK nao encontrada apos extracao")

    target = JDK_ROOT / "jdk-17"
    shutil.rmtree(target, ignore_errors=True)
    shutil.move(str(extracted[0]), target)

    java_exe = target / "bin" / "java.exe"
    if not java_exe.exists():
        raise RuntimeError(f"java.exe nao encontrado em {java_exe}")

    print(f"JDK pronto em: {target}")


if __name__ == "__main__":
    main()
