import groovy.transform.Field
import static com.sap.piper.Prerequisites.checkScript

@Field String STEP_NAME = getClass().getName()
@Field String METADATA_FILE = 'metadata/onapsisExecuteScan.yaml'

def call(Map parameters = [:], body) {
    final script = checkScript(this, parameters) ?: this
    List credentials = [[type: 'token', id: 'onapsisCredentialsId', env: ['PIPER_token']]]
    piperExecuteBin(parameters, STEP_NAME, METADATA_FILE, credentials)
}
