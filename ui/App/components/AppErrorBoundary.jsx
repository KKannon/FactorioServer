import React from "react";

class AppErrorBoundary extends React.Component {
    constructor(props) {
        super(props);
        this.state = {error: null};
    }

    static getDerivedStateFromError(error) {
        return {error};
    }

    componentDidCatch(error, details) {
        console.error('Frontend render failed', error, details);
    }

    render() {
        if (!this.state.error) return this.props.children;
        return (
            <main className="identity-loading p-6 text-center">
                <div>
                    <h1 className="text-xl font-bold mb-3">O painel encontrou um erro inesperado.</h1>
                    <p className="mb-4">A sessão continua segura. Recarregue a interface para tentar novamente.</p>
                    <button className="bg-orange text-black font-bold px-4 py-2" onClick={() => window.location.reload()}>
                        Recarregar painel
                    </button>
                </div>
            </main>
        );
    }
}

export default AppErrorBoundary;
