import {useEffect, useRef} from 'react';
import Drawflow from 'drawflow';
import 'drawflow/dist/drawflow.min.css';
import './style.css';

export default function App() {
    const containerRef = useRef<HTMLDivElement>(null);
    const editorRef = useRef<Drawflow | null>(null);

    useEffect(() => {
        if (!containerRef.current) return;

        const editor = new Drawflow(containerRef.current);
        editor.start();

        editor.addNode(
            'getOrders',
            1,
            1,
            200,
            100,
            'GetOrders',
            {},
            `<div><strong>GetOrders</strong></div>`,
            false,
        );

        editorRef.current = editor;
    }, []);

    const handleSave = () => {
        if (!editorRef.current) return;
        const json = editorRef.current.export();
        console.log('Saved:', json);
        // здесь будет fetch POST /api/settings
    };

    return (
        <div>
            <div ref={containerRef} style={{height: '80vh'}}/>
            <button onClick={handleSave} style={{margin: '8px'}}>Save</button>
        </div>
    );
}