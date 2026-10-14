import pathlib
import sys

# The tests import the CDK app's modules (countries.py, stage.py…) from infra/.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
