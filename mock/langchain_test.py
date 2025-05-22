from langchain_community.llms import Ollama
from langchain.callbacks.manager import CallbackManager
from langchain.callbacks.streaming_stdout import StreamingStdOutCallbackHandler
import os

# Configure the proxy URL and API key
PROXY_URL = "http://localhost:8080"  # Default proxy port
API_KEY = "test-api-key"  # Replace with your actual API key

# Create a custom Ollama instance that uses the proxy
llm = Ollama(
    base_url=PROXY_URL,
    model="gemma3:1b",  # or any other model you have available
    headers={
        "X-API-Key": API_KEY  # Add the API key header
    },
    callback_manager=CallbackManager([StreamingStdOutCallbackHandler()])
)

# Test the connection with a simple prompt
def test_ollama_proxy():
    try:
        response = llm.invoke("What is the capital of France?")
        print("\nResponse:", response)
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    test_ollama_proxy()
